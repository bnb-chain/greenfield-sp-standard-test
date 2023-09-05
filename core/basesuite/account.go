package basesuite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	sdkClient "github.com/bnb-chain/greenfield-go-sdk/client"
	sdkTypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/log"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/utils"
	"github.com/bnb-chain/greenfield/sdk/keys"
	spTypes "github.com/bnb-chain/greenfield/x/sp/types"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
)

type Account struct {
	KM            keys.KeyManager
	Addr          types.AccAddress
	PrimarySPAddr string
	SDKClient     sdkClient.Client
	Ctx           context.Context
	Secret        string
}

func NewAccount(greenfieldEndpoint, chainId, spAddress, secret string) *Account {
	c, err := sdkClient.New(chainId, greenfieldEndpoint, sdkClient.Option{GrpcDialOption: grpc.WithTransportCredentials(insecure.NewCredentials())})
	if err != nil {
		log.Errorf("sdk new client err: %v", err)
		panic(err)
	}

	var account *sdkTypes.Account
	accountName := utils.GetRandomAccountName()
	isMnemonic := strings.Contains(secret, " ")
	if secret == "" {
		account, secret, err = sdkTypes.NewAccount(accountName)
		log.Infof("new private key:%s", secret)
	} else if isMnemonic {
		account, err = sdkTypes.NewAccountFromMnemonic(accountName, secret)
	} else {
		account, err = sdkTypes.NewAccountFromPrivateKey(accountName, secret)
	}
	if err != nil {
		log.Errorf("sdk new account err: %v", err)
		return &Account{
			SDKClient:     c,
			PrimarySPAddr: spAddress,
			Ctx:           context.Background(),
			Secret:        secret,
		}
	}
	c.SetDefaultAccount(account)

	return &Account{
		KM:            account.GetKeyManager(),
		Addr:          account.GetAddress(),
		PrimarySPAddr: spAddress,
		SDKClient:     c,
		Ctx:           context.Background(),
		Secret:        secret,
	}
}
func (a *Account) SelectSP(primarySPAddr string) (*spTypes.StorageProvider, error) {
	providers, err := a.SDKClient.ListStorageProviders(context.Background(), false)
	if err != nil {
		return nil, err
	}
	spAddress, err := types.AccAddressFromHexUnsafe(primarySPAddr)
	if err != nil {
		return nil, err
	}
	primarySPAddr = spAddress.String()
	a.PrimarySPAddr = primarySPAddr
	for _, sp := range providers {
		log.Infof("sp operator address: %s, endpoint: %s, config sp address: %s", sp.OperatorAddress, sp.Endpoint, primarySPAddr)
		if sp.OperatorAddress == primarySPAddr {
			return &sp, nil
		}
	}

	return nil, fmt.Errorf("not funde this sp: %s", primarySPAddr)
}
func (a *Account) CreateBucket(bucketName string, opts *sdkTypes.CreateBucketOptions) (string, error) {
	if opts == nil {
		opts = &sdkTypes.CreateBucketOptions{}
	}
	bucketTx, err := a.SDKClient.CreateBucket(a.Ctx, bucketName, a.PrimarySPAddr, *opts)
	if err != nil {
		return "", err
	}
	_, err = a.SDKClient.WaitForTx(context.Background(), bucketTx)
	if err != nil {
		log.Errorf("wait tx: %s failed: %v", bucketTx, err)
		return "", err
	}

	return bucketTx, nil
}

func (a *Account) CreateAndUploadObject(bucketName, objectName string, fileSize uint64, visibility storageTypes.VisibilityType) error {
	opts := &sdkTypes.CreateObjectOptions{Visibility: visibility}
	_, res, file, err := a.CreateObjectAllSize(bucketName, objectName, fileSize, opts)
	if err != nil {
		return err
	}

	err = a.PutObject(bucketName, objectName, res, *file, nil)
	if err != nil {
		return err
	}

	return nil
}
func (a *Account) CreateObjectAllSize(bucketName, objectName string, fileSize uint64, opts *sdkTypes.CreateObjectOptions) (*storageTypes.ObjectInfo, string, *utils.File, error) {
	if opts == nil {
		opts = &sdkTypes.CreateObjectOptions{}
	}

	buffer := utils.GetFileBufferBySize(fileSize)
	txHash, err := a.SDKClient.CreateObject(a.Ctx, bucketName, objectName, bytes.NewReader(buffer.Bytes()), *opts)
	if err != nil {
		return nil, "", nil, err
	}

	_, err = a.SDKClient.WaitForTx(context.Background(), txHash)
	if err != nil {
		return nil, "", nil, err
	}
	info, err := a.SDKClient.HeadObject(context.Background(), bucketName, objectName)
	if err != nil {
		return nil, "", nil, err
	}
	return info.ObjectInfo, txHash, &utils.File{Reader: bytes.NewReader(buffer.Bytes()), Size: int64(buffer.Len())}, nil
}

func (a *Account) PutObject(bucketName, objectName, createObjectTx string, file utils.File, opts *sdkTypes.PutObjectOptions) error {
	if opts == nil {
		opts = &sdkTypes.PutObjectOptions{TxnHash: createObjectTx}
	}
	return a.SDKClient.PutObject(a.Ctx, bucketName, objectName, file.Size, file.Reader, *opts)
}

func (a *Account) IsObjectSealed(bucketName, objectName string) *storageTypes.ObjectInfo {
	i := 0
	for i < 360 {
		info, err := a.SDKClient.HeadObject(a.Ctx, bucketName, objectName)
		if err != nil {
			log.Errorf("HeadObject error: %v", err)
			continue
		}
		if info != nil && info.ObjectInfo != nil {
			log.Infof(" bucketName %s object %s  wait  %ds for seal， status: %s", bucketName, objectName, i, info.ObjectInfo.ObjectStatus.String())
			if info.ObjectInfo != nil && info.ObjectInfo.ObjectStatus == storageTypes.OBJECT_STATUS_SEALED {
				log.Infof("bucket: %s,object: %s is sealed", bucketName, objectName)
				return info.ObjectInfo
			}
		}
		time.Sleep(time.Second)
		i++
	}
	info, err := a.SDKClient.HeadObject(a.Ctx, bucketName, objectName)
	if err != nil {
		log.Errorf("HeadObject err: %v", err)
	}
	return info.ObjectInfo
}

func (a *Account) GetObject(bucketName, objectName string) (io.ReadCloser, sdkTypes.ObjectStat, error) {
	return a.SDKClient.GetObject(a.Ctx, bucketName, objectName, sdkTypes.GetObjectOptions{})
}
