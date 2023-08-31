package sp_standard_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	sdkClient "github.com/bnb-chain/greenfield-go-sdk/client"
	"github.com/bnb-chain/greenfield-sp-standard-test/config"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	httplib "github.com/bnb-chain/greenfield-common/go/http"
	sdkTypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/basesuite"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/log"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/utils"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
)

type SPFunctionalTestSuite struct {
	basesuite.BaseSuite
	suite.Suite
	spEndpointOptions *sdkTypes.EndPointOptions
}

func (s *SPFunctionalTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	config.CfgEnv.HttpHeaders["origin"] = "https://greenfield.bnbchain.org"
	config.CfgEnv.HttpHeaders["referer"] = "https://greenfield.bnbchain.org"
	s.spEndpointOptions = &sdkTypes.EndPointOptions{SPAddress: s.SPInfo.OperatorAddress, Endpoint: s.SPInfo.Endpoint}
}
func TestSPFunctional(t *testing.T) {
	suite.Run(t, new(SPFunctionalTestSuite))
}

func (s *SPFunctionalTestSuite) Test_00_SystemStatus() {
	statusUrl := fmt.Sprintf("%s/status", s.SPInfo.Endpoint)
	headers, statusInfo, err := utils.HttpGetWithHeaders(statusUrl, config.CfgEnv.HttpHeaders)
	s.NoError(err)
	log.Infof("statusInfo: %v", statusInfo)
	log.Infof("headers: %v", headers)
	s.True(utils.CheckHttpHeader(*headers, "https://greenfield.bnbchain.org", config.CfgEnv.HttpHeaders))
	blockSyncHeight := gjson.Get(statusInfo, "status.block_syncer_info.bs_block_height").Int()

	testAccount := s.TestAcc
	status, err := testAccount.SDKClient.GetStatus(context.Background())
	s.NoError(err)
	diff := status.SyncInfo.LatestBlockHeight - blockSyncHeight
	log.Infof("blockSyncHeight: %v, blockChain LatestBlockHeight: %d", blockSyncHeight, status.SyncInfo.LatestBlockHeight)
	s.True(diff < 30, "sp block sync height less block chain over 30 block")
}

func (s *SPFunctionalTestSuite) Test_01_UploadMultiSizeFile() {

	testAccount := s.TestAcc
	bucketName := utils.GetRandomBucketName()
	bucketTx, err := testAccount.CreateBucket(bucketName, nil)
	s.NoError(err)
	log.Infof("Created bucket: %s, txHash: %s", bucketName, bucketTx)
	bucket, err := testAccount.SDKClient.BuyQuotaForBucket(context.Background(), bucketName, 2*1024*1024*1024, sdkTypes.BuyQuotaOption{})
	s.NoError(err, "BuyQuotaForBucket error")
	log.Infof("BuyQuotaForBucket bucket: %s, txHash: %s", bucketName, bucket)

	testCases := []struct {
		name     string
		fileSize uint64
	}{
		{"Put 1B file", 1},
		{"Put 5.99MB file", 5*1024*1024 + 888},
		{"Put 16MB file", 16 * 1024 * 1024},
		{"Put 20MB file", 20 * 1024 * 1024},
		{"Put 256MB file", 256*1024*1024 + 12},
		{"Put 1G file", 1 * 1024 * 1024 * 1024},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			objectName := utils.GetRandomObjectName()

			_, tx, file, err := testAccount.CreateObjectAllSize(bucketName, objectName, tc.fileSize, nil)
			s.NoError(err)
			log.Infof("Created object - file size: %d bytes, endpoint: %s, tx: %v", tc.fileSize, s.SPInfo.Endpoint, tx)

			log.Infof("Uploading file - object: %s, bucket: %v", objectName, bucketName)
			err = testAccount.PutObject(bucketName, objectName, "", *file, nil)
			s.NoError(err, fmt.Sprintf("===Put failed - file size: %d, endpoint: %s===", tc.fileSize, s.SPInfo.Endpoint))
			log.Infof("Waiting for seal - object: %s, bucket: %v", objectName, bucketName)

			info := testAccount.IsObjectSealed(bucketName, objectName)
			s.Equal(info.ObjectStatus.String(), storageTypes.OBJECT_STATUS_SEALED.String(), fmt.Sprintf("===ObjectSealed failed - endpoint: %s, status: %s===", s.SPInfo.Endpoint, info.ObjectStatus.String()))
			time.Sleep(time.Second * 10)
			log.Infof("Downloading file - object: %s, bucket: %v", objectName, bucketName)
			quota, err := testAccount.SDKClient.GetBucketReadQuota(context.Background(), bucketName)
			s.NoError(err, "GetBucketReadQuota error")
			log.Infof("file Size: %d,BuyReadQuotaSize : %d, ReadConsumedSize: %d, SPFreeReadQuotaSize: %d", file.Reader.Size(), quota.ReadQuotaSize, quota.ReadConsumedSize, quota.SPFreeReadQuotaSize)
			fileDownLoad, info2, err := testAccount.GetObject(bucketName, objectName)
			s.NoError(err, fmt.Sprintf("===info2: %v, info: %v ", info2, info))

			hashA := md5.New()
			hashB := md5.New()
			_, err = io.Copy(hashB, fileDownLoad)
			s.NoError(err, "io copy hashB error")

			_, err = file.Reader.Seek(0, 0)
			s.NoError(err, "io Seek error")
			_, err = io.Copy(hashA, file.Reader)
			s.NoError(err, "io copy hashA error")

			fileHash := hex.EncodeToString(hashA.Sum(nil))
			downloadHash := hex.EncodeToString(hashB.Sum(nil))
			s.Equal(fileHash, downloadHash, "Downloading file hash check failed")
		})
	}
}
func (s *SPFunctionalTestSuite) Test_02_DeleteObjectBucket() {
	bucketName := utils.GetRandomBucketName()
	objectName := utils.GetRandomObjectName()
	fileSize := uint64(utils.RandInt64(1024, 10*1024))
	testAccount := s.TestAcc

	// Create bucket
	bucketTx, err := testAccount.CreateBucket(bucketName, nil)
	s.NoError(err)
	log.Infof("Created bucket: %s, txHash: %s", bucketName, bucketTx)

	// Create and upload object
	_, res, file, err := testAccount.CreateObjectAllSize(bucketName, objectName, fileSize, nil)
	s.NoError(err)
	err = testAccount.PutObject(bucketName, objectName, res, *file, nil)
	s.NoError(err)

	// Check if object is sealed
	objectInfo := testAccount.IsObjectSealed(bucketName, objectName)
	s.Equal(storageTypes.OBJECT_STATUS_SEALED.String(), objectInfo.ObjectStatus.String(), "object not sealed")

	// Delete object
	deleteObjectOption := sdkTypes.DeleteObjectOption{}
	deleteObjectTxHash, err := testAccount.SDKClient.DeleteObject(context.Background(), bucketName, objectName, deleteObjectOption)
	s.NoError(err)

	// Wait for delete transaction to complete
	txInfo, err := testAccount.SDKClient.WaitForTx(context.Background(), deleteObjectTxHash)
	s.NoError(err)
	s.True(txInfo.TxResult.Code == 0, txInfo)

	// Check if object info is nil after deletion
	objectInfo2, err := testAccount.SDKClient.HeadObject(context.Background(), bucketName, objectName)
	s.Error(err)
	s.Nil(objectInfo2, "after delete object cannot query object info")

	fileDownLoad, _, err := testAccount.GetObject(bucketName, objectName)
	s.Nil(fileDownLoad, "cannot load object")
	s.Error(err, "delete object cannot be get")

	deleteBucketOption := sdkTypes.DeleteBucketOption{}
	deleteBucketTxHash, err := testAccount.SDKClient.DeleteBucket(context.Background(), bucketName, deleteBucketOption)
	s.NoError(err, "delete object cannot be get")

	// Wait for delete bucket transaction to complete
	deleteBucketTxInfo, err := testAccount.SDKClient.WaitForTx(context.Background(), deleteBucketTxHash)
	s.NoError(err)
	s.True(deleteBucketTxInfo.TxResult.Code == 0, deleteBucketTxInfo)
}
func (s *SPFunctionalTestSuite) Test_03_CheckDownloadQuota() {
	bucketName := utils.GetRandomBucketName()
	objectName := utils.GetRandomObjectName()
	fileSize := uint64(utils.RandInt64(1024, 10*1024))
	testAccount := s.TestAcc

	// Create bucket
	bucketTx, err := testAccount.CreateBucket(bucketName, nil)
	s.NoError(err)
	log.Infof("Created bucket: %s, txHash: %s", bucketName, bucketTx)

	// Get storage price
	storagePrice, err := testAccount.SDKClient.GetStoragePrice(context.Background(), s.SPInfo.OperatorAddress)
	s.NoError(err)
	log.Infof("Quota price: %s", storagePrice.ReadPrice)

	// Get bucket read quota
	quotaInfo, err := testAccount.SDKClient.GetBucketReadQuota(context.Background(), bucketName)
	s.NoError(err)
	log.Infof("GetBucketReadQuota after create bucket: %v", quotaInfo)

	// Create and upload object
	_, res, file, err := testAccount.CreateObjectAllSize(bucketName, objectName, fileSize, nil)
	s.NoError(err)
	err = testAccount.PutObject(bucketName, objectName, res, *file, nil)
	s.True(err == nil)

	// Check if object is sealed
	objectInfo := testAccount.IsObjectSealed(bucketName, objectName)
	s.Equal(storageTypes.OBJECT_STATUS_SEALED.String(), objectInfo.ObjectStatus.String(), "object not sealed")

	// Check read quota records before downloading
	timesBefore := time.Now().UnixMilli()
	listBucketQuotaOps := sdkTypes.ListReadRecordOptions{StartTimeStamp: timesBefore}
	quotaRecord, err := testAccount.SDKClient.ListBucketReadRecord(context.Background(), bucketName, listBucketQuotaOps)
	s.NoError(err)
	s.True(len(quotaRecord.ReadRecords) == 0)

	// Download object
	reader, _, err := testAccount.GetObject(bucketName, objectName)
	s.NoError(err)
	fileBytes, err := io.ReadAll(reader)
	s.NoError(err)
	s.Equal(len(fileBytes), int(fileSize))

	// Check if read quota record is updated
	timeAfterDownload := time.Now().UnixMilli()
	listBucketQuotaOps.StartTimeStamp = timeAfterDownload
	quotaRecord1, err := testAccount.SDKClient.ListBucketReadRecord(context.Background(), bucketName, listBucketQuotaOps)
	s.NoError(err)
	s.True(len(quotaRecord1.ReadRecords) == 1)
	s.Equal(fileSize, quotaRecord1.ReadRecords[0].ReadSize)
}

func (s *SPFunctionalTestSuite) Test_04_VerifyAuth() {
	bucketName := utils.GetRandomBucketName()
	fileSize := uint64(utils.RandInt64(1024, 3*1024))
	testAccountA := s.TestAcc
	testAccountB := s.RootAcc

	// Create bucket using testAccountA
	bucketTx, err := testAccountA.CreateBucket(bucketName, nil)
	s.NoError(err)
	log.Infof("Created bucket: %s, txHash: %s", bucketName, bucketTx)

	objectName0 := utils.GetRandomObjectName()
	_, _, _, err = testAccountB.CreateObjectAllSize(bucketName, objectName0, fileSize, nil)
	s.Error(err, "testAccountB should not be able to create object in testAccountA's bucket")

	objectName1 := utils.GetRandomObjectName()
	_, createObjectTxHash0, file, err := testAccountA.CreateObjectAllSize(bucketName, objectName1, fileSize, nil)
	s.NoError(err)
	err = testAccountB.PutObject(bucketName, objectName1, createObjectTxHash0, *file, nil)
	s.Error(err, "testAccountB should not be able to put object in testAccountA's bucket")

	objectName2 := utils.GetRandomObjectName()
	_, createObjectTxHash1, file, err := testAccountA.CreateObjectAllSize(bucketName, objectName2, fileSize, nil)
	s.NoError(err)
	err = testAccountA.PutObject(bucketName, objectName2, createObjectTxHash1, *file, nil)
	s.NoError(err)

	objectInfo := testAccountB.IsObjectSealed(bucketName, objectName2)
	s.Equal(storageTypes.OBJECT_STATUS_SEALED.String(), objectInfo.ObjectStatus.String(), "Object not sealed")

	// Attempt to get object from testAccountB (should fail)
	fileDownLoad, _, err := testAccountB.GetObject(bucketName, objectName2)
	s.Nil(fileDownLoad, "testAccountB should not be able to get object from testAccountA's bucket")
	s.Error(err, "testAccountB should not be able to get object from testAccountA's bucket")

	// Get object from testAccountA
	fileDownLoad, _, err = testAccountA.GetObject(bucketName, objectName2)
	s.NoError(err)
	s.NotEmpty(fileDownLoad)
}

func (s *SPFunctionalTestSuite) Test_05_ListUserBucketObject() {
	testAccount := s.TestAcc
	// List buckets for the testAccount
	listBuckets, err := testAccount.SDKClient.ListBuckets(context.Background(), sdkTypes.ListBucketsOptions{EndPointOptions: s.spEndpointOptions}) //
	s.NoError(err)
	s.NotEmpty(listBuckets.Buckets)
	log.Infof("List users: %s buckets: %v", testAccount.Addr.String(), listBuckets.Buckets)

	// List objects for the first bucket
	bucketName := listBuckets.Buckets[0].BucketInfo.BucketName
	listObjects, err := testAccount.SDKClient.ListObjects(context.Background(), bucketName, sdkTypes.ListObjectsOptions{EndPointOptions: s.spEndpointOptions})
	s.NoError(err)
	log.Infof("List users: %s objects: %v", testAccount.Addr.String(), listObjects.Objects)
}

func (s *SPFunctionalTestSuite) Test_06_GetNonce() {
	userAddress := s.TestAcc.Addr.String()
	headers := config.CfgEnv.HttpHeaders
	headers["X-Gnfd-User-Address"] = userAddress
	headers["X-Gnfd-App-Domain"] = "https://greenfield.bnbchain.org/"

	respHeader, response, err := utils.GetNonce(s.SPInfo.Endpoint, headers)
	log.Debugf("GetNonce response: %v", response)
	log.Infof("respHeader: %v", respHeader)

	s.NoError(err, "call /auth/request_nonce error")
	s.NotEmpty(response)
	s.True(strings.Contains(response, "NextNonce"))
	s.True(utils.CheckHttpHeader(*respHeader, "https://greenfield.bnbchain.org", config.CfgEnv.HttpHeaders))

}

func (s *SPFunctionalTestSuite) Test_08_BucketsByIdsObjectsByIds() {
	testAccount := s.TestAcc
	listBuckets, err := testAccount.SDKClient.ListBuckets(context.Background(), sdkTypes.ListBucketsOptions{EndPointOptions: s.spEndpointOptions})
	s.NoError(err)
	s.NotEmpty(listBuckets.Buckets)
	log.Infof("list users: %s , buckets length: %v", testAccount.Addr.String(), len(listBuckets.Buckets))

	bucketsId := []uint64{listBuckets.Buckets[0].BucketInfo.Id}
	response0, err := testAccount.SDKClient.ListBucketsByBucketID(context.Background(), bucketsId, *s.spEndpointOptions)
	log.Infof("ListBucketsByBucketID: %v", response0.Buckets[0])
	s.NoError(err, "call buckets-query error")
	s.NotEmpty(response0.Buckets)
	objectId := uint64(0)
	for _, bucket := range listBuckets.Buckets {
		bucketName := bucket.BucketInfo.BucketName
		listObjects, _ := testAccount.SDKClient.ListObjects(context.Background(), bucketName, sdkTypes.ListObjectsOptions{EndPointOptions: s.spEndpointOptions})
		log.Infof("list users: %s objects length: %v", testAccount.Addr.String(), len(listObjects.Objects))
		if len(listObjects.Objects) != 0 {
			objectId = listObjects.Objects[0].ObjectInfo.Id
			break
		}
	}
	objectIds := []uint64{objectId}
	response, err := testAccount.SDKClient.ListObjectsByObjectID(context.Background(), objectIds, sdkTypes.EndPointOptions{})
	log.Infof("ListObjectsByObjectID: %v", response.Objects[0])
	s.NoError(err, "call objects-query error")
	s.NotEmpty(response)
}

func (s *SPFunctionalTestSuite) Test_09_VerifySPPrice() {
	testAccount := s.TestAcc
	spPriceInfo, err := testAccount.SDKClient.GetStoragePrice(context.Background(), s.SPInfo.OperatorAddress)
	s.NoError(err)

	storePrice, _ := spPriceInfo.StorePrice.Float64()
	readPrice, _ := spPriceInfo.ReadPrice.Float64()
	log.Infof("Read price: %v, Store price: %v, freeQuota: %d", readPrice, storePrice, spPriceInfo.FreeReadQuota)
	s.NotZero(storePrice, "Store price is 0")
	s.NotZero(readPrice, "Read price is 0")
}

func (s *SPFunctionalTestSuite) Test_10_ListGroupByNameAndPrefix() {
	// group name start "prefix", contain "name"
	name := "x"
	prefix := "t"
	testAccount := s.TestAcc
	listGroupByNameAndPrefix, err := testAccount.SDKClient.ListGroup(context.Background(), name, prefix, sdkTypes.ListGroupsOptions{EndPointOptions: s.spEndpointOptions})
	log.Infof("listGroupByNameAndPrefix: %v", listGroupByNameAndPrefix)
	s.NoError(err, "ListGroupsByNameAndPrefix error")
}

func (s *SPFunctionalTestSuite) Test_11_UniversalEndpoint() {
	testAccount := s.TestAcc
	bucketName := utils.GetRandomBucketName()
	publicObjectName := utils.GetRandomObjectName()
	privateObjectName := utils.GetRandomObjectName()
	fileSize := uint64(utils.RandInt64(1024, 10*1024))

	// Create bucket
	bucketTx, err := testAccount.CreateBucket(bucketName, nil)
	s.NoError(err)
	log.Infof("Created bucket: %s, txHash: %s", bucketName, bucketTx)

	// Create and upload private object
	err = testAccount.CreateAndUploadObject(bucketName, privateObjectName, fileSize, storageTypes.VISIBILITY_TYPE_PRIVATE)
	s.NoError(err)

	// Check if private object is sealed
	objectInfo := testAccount.IsObjectSealed(bucketName, privateObjectName)
	s.Require().Equal(storageTypes.OBJECT_STATUS_SEALED, objectInfo.ObjectStatus, "private object not sealed")

	// Create and upload public object
	err = testAccount.CreateAndUploadObject(bucketName, publicObjectName, fileSize, storageTypes.VISIBILITY_TYPE_PUBLIC_READ)
	s.NoError(err)

	// Check if public object is sealed
	objectInfo2 := testAccount.IsObjectSealed(bucketName, publicObjectName)
	s.Require().Equal(storageTypes.OBJECT_STATUS_SEALED, objectInfo2.ObjectStatus, "public object not sealed")

	publicUniversalEndpoint := fmt.Sprintf("%s/view/%s/%s", s.SPInfo.Endpoint, bucketName, publicObjectName)
	privateUniversalEndpoint := fmt.Sprintf("%s/download/%s/%s", s.SPInfo.Endpoint, bucketName, privateObjectName)
	log.Infof("publicUniversalEndpoint: %s", publicUniversalEndpoint)
	log.Infof("privateUniversalEndpoint: %s", privateUniversalEndpoint)
	time.Sleep(5 * time.Second)
	// case 1: access universal endpoint from non-browser;
	headers := config.CfgEnv.HttpHeaders

	respHeader, response, err := utils.HttpGetWithHeaders(publicUniversalEndpoint, headers)
	log.Debugf(" publicUniversalEndpoint Response is :%v, error is %v", response, err)
	s.True(len(response) == int(fileSize))
	s.True(utils.CheckHttpHeader(*respHeader, "https://greenfield.bnbchain.org", config.CfgEnv.HttpHeaders))

	// case 2: access universal endpoint from public object
	headers["User-Agent"] = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/114.0" //
	_, response, err = utils.HttpGetWithHeaders(publicUniversalEndpoint, headers)
	log.Debugf("publicUniversalEndpoint response: %s", response)
	s.NoError(err)
	s.True(!strings.Contains(response, "error"))

	// case 3: access universal endpoint without auth string from browser; expect to get a build-in dapp HTML
	_, response, err = utils.HttpGetWithHeaders(privateUniversalEndpoint, headers)
	log.Debugf("access universal endpoint without auth string, from browser,  Response is :%v, error is %v", response, err)
	s.True(strings.Contains(response, "<!doctype html><html")) // <!doctype html><html....

	// case 4: use user's private key to make a wallet personal sign, and append the signature to the private universal endpoint
	const GnfdBuiltInDappSignedContentTemplate = "Sign this message to access the file:\n%s\nThis signature will not cost you any fees.\nExpiration Time: %s"
	ExpiryDateFormat := "2006-01-02T15:04:05Z"
	expiryStr := time.Now().Add(time.Minute * 4).Format(ExpiryDateFormat)
	signedMsg := fmt.Sprintf(GnfdBuiltInDappSignedContentTemplate, "gnfd://"+bucketName+"/"+privateObjectName, expiryStr)
	signedMsgHash := accounts.TextHash([]byte(signedMsg))

	sig, _ := s.TestAcc.KM.Sign(signedMsgHash)
	signString := utils.ConvertToString(sig)

	universalEndpointWithPersonalSig := fmt.Sprintf("%s?X-Gnfd-Expiry-Timestamp=%s&signature=%s", privateUniversalEndpoint, expiryStr, signString)
	log.Debugf("universalEndpointWithPersonalSig is: " + universalEndpointWithPersonalSig)
	_, response, err = utils.HttpGetWithHeaders(universalEndpointWithPersonalSig, headers)
	s.NoError(err, "universalEndpointWithPersonalSig error")
	log.Debugf("access universal endpoint with auth string, from browser,  Response is :%v, error is %v", response, err)
	s.True(len(response) == int(fileSize))

}
func (s *SPFunctionalTestSuite) Test_12_OffChainAuth() {
	defaultAcct, _ := s.TestAcc.SDKClient.GetDefaultAccount()
	offChainClient, err := sdkClient.New(config.CfgEnv.GreenfieldChainId, config.CfgEnv.GreenfieldEndpoint, sdkClient.Option{
		DefaultAccount: defaultAcct,
		OffChainAuthOption: &sdkClient.OffChainAuthOption{
			Seed:                 "test_seed",
			Domain:               "https://test.domain.com",
			ShouldRegisterPubKey: true,
		}, GrpcDialOption: grpc.WithTransportCredentials(insecure.NewCredentials())})
	s.TestAcc.SDKClient = offChainClient

	if err != nil {
		log.Errorf("sdk new client err: %v", err)
		panic(err)
	}

	bucketName := utils.GetRandomBucketName()
	objectName := utils.GetRandomObjectName()
	fileSize := uint64(utils.RandInt64(1024, 10*1024))
	testAccount := s.TestAcc

	// Create bucket
	bucketTx, err := testAccount.CreateBucket(bucketName, nil)
	s.NoError(err)
	log.Infof("Created bucket: %s, txHash: %s", bucketName, bucketTx)

	// Create and upload object
	_, res, file, err := testAccount.CreateObjectAllSize(bucketName, objectName, fileSize, nil)
	s.NoError(err)
	err = testAccount.PutObject(bucketName, objectName, res, *file, nil)
	s.NoError(err)

	// Check if object is sealed
	objectInfo := testAccount.IsObjectSealed(bucketName, objectName)
	s.Equal(storageTypes.OBJECT_STATUS_SEALED, objectInfo.ObjectStatus, "object not sealed")
	time.Sleep(5 * time.Second)

	// download file in a pre-signed way by calling getObject API
	getObjectEndpoint := fmt.Sprintf("%s/%s/%s", s.SPInfo.Endpoint, bucketName, objectName)

	ExpiryDateFormat := "2006-01-02T15:04:05Z"
	expiryStr := time.Now().Add(time.Minute * 4).Format(ExpiryDateFormat)

	getObjectEndpointWithPresignedParams := fmt.Sprintf("%s?X-Gnfd-Expiry-Timestamp=%s&X-Gnfd-User-Address=%s&X-Gnfd-App-Domain=%s", getObjectEndpoint, expiryStr, defaultAcct.GetAddress().String(), "https://test.domain.com")
	log.Debugf("getObjectEndpointWithPreSignedParams is: " + getObjectEndpointWithPresignedParams)
	req, err := http.NewRequest(http.MethodGet, getObjectEndpointWithPresignedParams, nil)
	s.NoError(err, req)
	unsignedMsg := httplib.GetMsgToSignInGNFD1AuthForPreSignedURL(req)
	authStr := testAccount.SDKClient.OffChainAuthSign(unsignedMsg)
	getObjectEndpointWithPresignedParams = getObjectEndpointWithPresignedParams + "&Authorization=" + url.QueryEscape(authStr)
	log.Debugf("getObjectEndpointWithPresignedParams is %s", getObjectEndpointWithPresignedParams)

	_, fileDownLoadStr, err := utils.HttpGetWithHeaders(getObjectEndpointWithPresignedParams, make(map[string]string))
	s.NoError(err, "getObjectEndpointWithPresignedParams error")
	log.Debugf("access getObjectEndpoint with auth preSignedURL, from browser,  Response is :%v, error is %v", fileDownLoadStr, err)
	s.True(len(fileDownLoadStr) == int(fileSize), fileDownLoadStr)
}
