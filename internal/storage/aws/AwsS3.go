// +build !noaws
// +build !awsmock

package aws

import (
	"Gokapi/internal/models"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"net/http"
	"os"
	"time"
)

// IsAvailable is true if Gokapi has been compiled with AWS support or the API is being mocked
const IsAvailable = true

// IsMockApi is true if the API is being mocked and therefore can only be used for testing purposes
const IsMockApi = false

// IsCredentialProvided returns true if all credentials are provided
func IsCredentialProvided(checkIfValid bool) bool {
	requiredKeys := []string{"GOKAPI_AWS_BUCKET", "AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"}
	for _, key := range requiredKeys {
		if !isValidEnv(key) {
			return false
		}
	}
	if checkIfValid {
		return checkIfValidLogin()
	}
	return true
}

func checkIfValidLogin() bool {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess)
	_, err := svc.Config.Credentials.Get()
	if err != nil {
		fmt.Println("WARNING: AWS S3 login not successful: " + err.Error())
		return false
	}
	fmt.Println("AWS S3 login successful")
	return true
}

func isValidEnv(key string) bool {
	val, ok := os.LookupEnv(key)
	return ok && val != ""
}

// Upload uploads a file to AWS
func Upload(input io.Reader, file models.File) (string, error) {
	sess := session.Must(session.NewSession())
	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
		Body:   input,
	})
	if err != nil {
		return "", err
	}
	return result.Location, nil
}

// Download downloads a file from AWS
func Download(writer io.WriterAt, file models.File) (int64, error) {
	sess := session.Must(session.NewSession())
	downloader := s3manager.NewDownloader(sess)

	size, err := downloader.Download(writer, &s3.GetObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

// RedirectToDownload creates a presigned link that is valid for 15 seconds and redirects the
// client to this url
func RedirectToDownload(w http.ResponseWriter, r *http.Request, file models.File) error {
	sess := session.Must(session.NewSession())
	s3svc := s3.New(sess)

	req, _ := s3svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket:                     aws.String(file.AwsBucket),
		Key:                        aws.String(file.SHA256),
		ResponseContentDisposition: aws.String("filename=" + file.Name),
	})

	url, err := req.Presign(15 * time.Second)
	if err != nil {
		return err
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

// FileExists returns true if the object is stored in S3
func FileExists(file models.File) (bool, error) {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	_, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
	})

	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok {
			if aerr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// DeleteObject deletes a file from S3
func DeleteObject(file models.File) (bool, error) {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(file.AwsBucket),
		Key:    aws.String(file.SHA256),
	})

	if err != nil {
		return false, err
	}
	return true, nil
}