# greenfield-sp-standard-test
## Description
The Greenfield Storage Provider Standard Test is a testing tool used to verify if the SP meets the standards for joining the Greenfield network. It integrates functional verification and performance verification testing.

## Disclaimer
**The software and related documentation are under active development, all subject to potential future change without
notification and not ready for production use. The code and security audit have not been fully completed and not ready
for any bug bounty. We advise you to be careful and experiment on the network at your own risk. Stay safe out there.**

## Usage

Clone the repository.
```shell
 git clone https://github.com/node-real/greenfield-sp-standard-test.git
 cd greenfield-sp-standard-test
 go mod tidy
```
1. Configure the SP address and the private key and test account in the configuration file ./config/config.yaml.
2. Run the functional verification tests by executing the following command:
    ```shell
     go test -v -run TestSPFunctional -timeout 30m
    ```
3. Run the performance verification tests by executing the following command:
    ```shell
     go test -v -run TestSPLoad -timeout 60m
    ```

## Functional verification(func_test.go)
The functional verification tests ensure that the storage provider supports the functionalities required to meet the standard. The following functionalities are verified:

- Test_00_UploadMultiSizeFile：
   - Upload and download of files of different sizes.
- Test_01_DeleteObjectBucket：
   - Deletion of buckets and objects.
- Test_02_CheckDownloadQuota:
   - Quota check for downloads.
- Test_03_VerifySPPrice:
   - Verify SP price is not zero.
- Test_04_VerifyAuth:
   - Permission check for file upload and download.
- Test_05_ListUserBucketObject:
   - Listing user buckets and objects.
- Test_06_GetNonce:
   - Query the current user account key record.
- Test_08_BucketsByIdsObjectsByIds:
   - Listing user bucket and object by ID.
- Test_09_ListGroupByNameAndPrefix:
   - Query a list of group by given prefix/name.
- Test_10_UpdateAccountKey:
   - Update the current user account key record in SP.
- Test_11_UniversalEndpoint:
   - Public and private universal Endpoint verification.
- Test_12_OffChainAuth:
   - Verify off-chain authentication specification.


## Performance verification(load_test.go)
The performance verification tests confirm if the storage provider's upload and download functions work properly under concurrent conditions. The following parameters can be configured in the config/config.yaml file:
```yaml
# performance test params
TPS: 2
# Number of users, If the user.csv file does not exist or the number of accounts in it is less than the configured value, new account private keys will be automatically generated and saved in the user.csv file
NumUsers: 10
# performance test duration(in seconds)
Duration: 60
Upload: true
Download: true
# File size (in bytes) random ~10kb (FileSize + random(10*1024))
FileSize: 1024
```

During the execution of the performance tests, the generated test private keys and the information of created buckets and objects will be automatically saved in user.csv, buckets.csv, and objects.csv. If you need to reset and rerun the tests, you can manually delete these files.




