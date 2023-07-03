package sp_standard_test

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/stretchr/testify/suite"
	"go.uber.org/ratelimit"

	"github.com/bnb-chain/greenfield-sp-standard-test/config"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/basesuite"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/log"
	"github.com/bnb-chain/greenfield-sp-standard-test/core/utils"
	storageTypes "github.com/bnb-chain/greenfield/x/storage/types"
)

var (
	usersFileName   = "users.csv"
	bucketsFileName = "buckets.csv"
	objectsFileName = "objects.csv"
)

type SPLoadTestSuite struct {
	basesuite.BaseSuite
	suite.Suite
	loadTestAccount utils.ParametersIterator
	loadTestBuckets map[string][]string
	loadTestObjects map[string][]string
	bucketsWriter   *csv.Writer
	objectsWriter   *csv.Writer
}

func (s *SPLoadTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	pwd, _ := os.Getwd()
	usersFilePath := path.Join(pwd, usersFileName)
	usersData := utils.GetParamFromCSV(usersFilePath, false)
	userCount := config.CfgEnv.NumUsers
	//More than 100 can cause long loading times
	if userCount > 100 {
		userCount = 100
	}
	loadTestAccounts := make([]*basesuite.Account, userCount)
	newUsers := make([][]string, userCount)
	for i := 0; i < userCount; i++ {
		if i < len(usersData) {
			loadTestAccounts[i] = basesuite.NewAccount(config.CfgEnv.GreenfieldEndpoint, config.CfgEnv.GreenfieldChainId, config.CfgEnv.SPAddr, usersData[i][1])
		} else {
			loadTestAccounts[i] = basesuite.NewAccount(config.CfgEnv.GreenfieldEndpoint, config.CfgEnv.GreenfieldChainId, config.CfgEnv.SPAddr, "")
			//save new user to file
			newUserAddr := loadTestAccounts[i].Addr.String()
			newUsers[i] = []string{newUserAddr, loadTestAccounts[i].Secret}
		}
	}
	if len(newUsers) > 0 {
		err := utils.WriteToCSV(newUsers, usersFilePath)
		if err != nil {
			log.Errorf("Error writing: %v", err)
		}
	}
	s.InitAccountsBNBBalance(loadTestAccounts, 1e16)
	s.loadTestAccount = utils.NewParametersIterator(loadTestAccounts)
	//	 get buckets and objects test data
	bucketsFilePath := path.Join(pwd, bucketsFileName)
	objectsFilePath := path.Join(pwd, objectsFileName)
	bucketsCSVFile, err := os.OpenFile(bucketsFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Failed to open buckets file: %v", err)
	}
	objectsCSVFile, err := os.OpenFile(objectsFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Failed to open objects file: %v", err)
	}

	s.bucketsWriter = csv.NewWriter(bucketsCSVFile)
	s.objectsWriter = csv.NewWriter(objectsCSVFile)
	r0 := csv.NewReader(bucketsCSVFile)
	r1 := csv.NewReader(objectsCSVFile)
	bucketsData, _ := r0.ReadAll()
	objectsData, _ := r1.ReadAll()
	loadTestBuckets := map[string][]string{}
	loadTestObjects := map[string][]string{}
	for _, datum := range bucketsData {
		userAddr := datum[0]
		bucketName := datum[1]
		//Avoid duplicate buckets
		if _, ok := loadTestBuckets[userAddr]; ok && !(utils.IsContain(loadTestBuckets[userAddr], bucketName)) {
			loadTestBuckets[userAddr] = append(loadTestBuckets[userAddr], bucketName)
		}
	}
	for _, datum := range objectsData {
		bucketName := datum[0]
		objectName := datum[1]
		// Avoid duplicate objects
		if _, ok := loadTestObjects[bucketName]; !(ok && utils.IsContain(loadTestObjects[bucketName], objectName)) {
			loadTestObjects[bucketName] = append(loadTestObjects[bucketName], objectName)
		}
	}
	s.loadTestBuckets = loadTestBuckets
	s.loadTestObjects = loadTestObjects
}
func TestSPLoad(t *testing.T) {
	suite.Run(t, new(SPLoadTestSuite))
}

func (s *SPLoadTestSuite) Test_upload_download() {
	fileSize := config.CfgEnv.FileSize
	rateLimit := config.CfgEnv.TPS
	if rateLimit > 10 {
		panic("===TPS > 10,  not supported===")
	}
	duration := config.CfgEnv.Duration
	expired := setupTimer(time.Duration(duration) * time.Second)
	rateLimiter := ratelimit.New(rateLimit)

	var wg sync.WaitGroup
	var lock sync.Mutex
	pool, _ := ants.NewPool(rateLimit + 2)
	// print test results
	s.PrintTestResult()
	for {
		// exit time
		if *expired {
			pool.Release()
			break
		}
		rateLimiter.Take()
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := pool.Submit(func() {
				testAccount := s.loadTestAccount.Next().(*basesuite.Account)
				testUserAddr := testAccount.Addr.String()
				r := rand.Int() % 2
				if config.CfgEnv.Upload && r == 0 {
					objectName := utils.GetRandomObjectName()
					bucketName := utils.GetRandomBucketName()

					// Controller user bucket number < 10
					if _, ok := s.loadTestBuckets[testUserAddr]; ok && len(s.loadTestBuckets[testUserAddr]) >= 10 {
						bucketName = s.loadTestBuckets[testUserAddr][rand.Intn(len(s.loadTestBuckets[testUserAddr]))]

					} else {
						createBucketTime := time.Now()
						bucketTx, err := testAccount.CreateBucket(bucketName, nil)
						log.Infof("User: %s, creating bucket: %s, bucketTx: %s", testUserAddr, bucketName, bucketTx)

						s.NewOutput("CreateBucket", err, createBucketTime)
						if err != nil {
							log.Errorf("CreateBucket err: %v", err)
							return
						}
						lock.Lock()
						s.loadTestBuckets[testUserAddr] = append(s.loadTestBuckets[testUserAddr], bucketName)
						err = s.bucketsWriter.Write([]string{testUserAddr, bucketName})
						if err != nil {
							log.Errorf("Write Bucket err: %v", err)
						}
						s.bucketsWriter.Flush()
						lock.Unlock()
					}
					// random 10kb file size for testing
					randomSize := uint64(rand.Int63n(1024 * 10))
					createObjectTime := time.Now()
					_, createObjectTx, file, err := testAccount.CreateObjectAllSize(bucketName, objectName, fileSize+randomSize, nil)
					s.NewOutput("CreateObject", err, createObjectTime)
					log.Infof("CreateObject objectName: %s, bucketName: %s, createObjectTx: %s", objectName, bucketName, createObjectTx)
					if err != nil {
						log.Errorf("CreateObjectAllSize err: %v", err)
						return
					}
					putObjectTime := time.Now()
					err = testAccount.PutObject(bucketName, objectName, "", *file, nil)
					s.NewOutput("PutObject", err, putObjectTime)
					log.Infof("PutObject objectName: %s, bucketName: %s", objectName, bucketName)
					if err != nil {
						log.Errorf("PutObject failed err: %v", err)
						return
					}
					objectSealTime := time.Now()
					info := testAccount.IsObjectSealed(bucketName, objectName)
					if info.ObjectStatus != storageTypes.OBJECT_STATUS_SEALED {
						err = fmt.Errorf("seal object failed: %s", objectName)
					}
					s.NewOutput("SealObject", err, objectSealTime)
					if err == nil {
						// add to test parameters
						lock.Lock()
						s.loadTestObjects[bucketName] = append(s.loadTestObjects[bucketName], objectName)
						err = s.objectsWriter.Write([]string{bucketName, objectName})
						if err != nil {
							log.Errorf("Write Bucket err: %v", err)
						}
						s.objectsWriter.Flush()
						lock.Unlock()
					}
				}
				if config.CfgEnv.Download && r == 1 {
					bucketName := ""
					objectName := ""

					if bucketList, ok := s.loadTestBuckets[testUserAddr]; ok {
						bucketName = bucketList[rand.Intn(len(bucketList))]

						if objectList, ok := s.loadTestObjects[bucketName]; ok {
							objectName = objectList[rand.Intn(len(objectList))]
						}
					}

					if bucketName == "" || objectName == "" {
						log.Errorf("No object available for download test, need to upload first")
						return
					}

					log.Infof("Downloading object: %s, bucket name: %s", objectName, bucketName)
					objectDownloadTime := time.Now()
					var buf bytes.Buffer
					fileDownLoadReader, _, err := testAccount.GetObject(bucketName, objectName)
					s.NewOutput("DownloadObject", err, objectDownloadTime)

					if err != nil {
						log.Errorf("Download object error: %v", err)
						return
					}

					_, err = io.Copy(&buf, fileDownLoadReader)
					if err != nil {
						log.Errorf("Download object copy error: %v", err)
					}
				}

			})
			if err != nil {
				log.Errorf("pool submit err: %v", err)
				return
			}
		}()

	}
	wg.Wait()

}

func setupTimer(dur time.Duration) *bool {
	t := time.NewTimer(dur)
	expired := false
	go func() {
		<-t.C
		expired = true
	}()
	return &expired
}
