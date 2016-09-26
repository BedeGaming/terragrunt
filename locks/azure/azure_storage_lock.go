package azure

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"time"

	"crypto/tls"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/gruntwork-io/terragrunt/locks"
	"github.com/gruntwork-io/terragrunt/util"
	"github.com/satori/uuid"
)

// used as suffix for backup files
const backupTimeFormat = time.RFC3339

// StorageLock provides a lock backed by Azure Storage
type StorageLock struct {
	StorageAccountName string
	ContainerName      string
	Key                string
	Backup             bool
}

// New is the factory function for StorageLock
func New(conf map[string]interface{}) (locks.Lock, error) {
	if _, ok := conf["storage_account_name"]; !ok {
		return nil, fmt.Errorf("storage_account_name must be set")
	}

	if _, ok := conf["container_name"]; !ok {
		return nil, fmt.Errorf("container_name must be set")
	}

	if _, ok := conf["key"]; !ok {
		return nil, fmt.Errorf("key must be set")
	}

	lock := &StorageLock{
		StorageAccountName: conf["storage_account_name"].(string),
		ContainerName:      conf["container_name"].(string),
		Key:                conf["key"].(string),
	}

	if backup, ok := conf["backup"]; ok {
		lock.Backup = backup.(bool)
	}

	return lock, nil
}

// AcquireLock attempts to create a Blob in the Storage Container
func (lock *StorageLock) AcquireLock() error {
	util.Logger.Printf("azure.StorageLock: attempting to acquire lock for Blob key %s", lock.Key)

	client, err := lock.createStorageClient()
	if err != nil {
		return err
	}

	exists, err := client.BlobExists(lock.ContainerName, lock.Key)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("lock blob does not exist")
	}

	proposedLeaseID := uuid.NewV4().String()
	_, err = client.AcquireLease(lock.ContainerName, lock.Key, -1, proposedLeaseID)
	if err != nil {
		return err
	}

	os.Setenv("ARM_LEASE_ID", proposedLeaseID)

	util.Logger.Printf("azure.StorageLock: lock acquired!")

	if lock.Backup {
		util.Logger.Printf("azure.StorageLock: backing up")

		if err := lock.backupBlob(client); err != nil {
			return fmt.Errorf("unable to backup state: %s", err)
		}
	}

	return nil
}

// ReleaseLock attempts to delete the Blob in the Storage Container
func (lock *StorageLock) ReleaseLock() error {
	util.Logger.Printf("azure.StorageLock: attempting to release lock for Blob key %s", lock.Key)

	client, err := lock.createStorageClient()
	if err != nil {
		return err
	}

	_, err = client.BreakLeaseWithBreakPeriod(lock.ContainerName, lock.Key, 0)
	if err != nil {
		return err
	}

	os.Setenv("ARM_LEASE_ID", "")

	util.Logger.Printf("azure.StorageLock: lock released!")
	return nil
}

// String returns a description of this lock
func (lock *StorageLock) String() string {
	return fmt.Sprintf("azure.StorageLock for state file %s", lock.Key)
}

// createStorageClient creates a new Blob Storage Client from the Azure SDK
// returns and error if ARM_ACCESS_KEY is empty
func (lock *StorageLock) createStorageClient() (*storage.BlobStorageClient, error) {
	accessKey := os.Getenv("ARM_ACCESS_KEY")
	if accessKey == "" {
		return nil, fmt.Errorf("ARM_ACCESS_KEY environment variable must be set")
	}

	client, err := storage.NewBasicClient(lock.StorageAccountName, accessKey)
	if err != nil {
		return nil, err
	}

	client.HTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	blobClient := client.GetBlobService()
	return &blobClient, nil
}

func (lock *StorageLock) backupBlob(client *storage.BlobStorageClient) error {
	r, err := client.GetBlob(lock.ContainerName, lock.Key)
	if err != nil {
		return err
	}
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	backupName := fmt.Sprintf("%s.%s", lock.Key, time.Now().Format(backupTimeFormat))
	buf := bytes.NewBuffer(b)
	return client.CreateBlockBlobFromReader(lock.ContainerName, backupName, uint64(len(b)), buf, nil)
}
