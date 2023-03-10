package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "client_app/docs" // docs is generated by Swag CLI, you have to import it.
)

// gin-swagger middleware
// swagger embed files

func fileToBase64(filename string) (string, int64) {
	// Open file on disk.
	f, _ := os.Open(filename)
	stat, _ := f.Stat()

	// Read entire JPG into byte slice.
	reader := bufio.NewReader(f)
	content, _ := ioutil.ReadAll(reader)

	// Encode as base64.
	encoded := base64.StdEncoding.EncodeToString(content)

	// Print encoded data to console.
	// ... The base64 image can be used as a data URI in a browser.
	return encoded, stat.Size()
}

func getMinio() (*minio.Client, error) {
	endpoint := "127.0.0.1:9000"
	accessKeyID := "EafiiIyEVsF2vbn4"
	secretAccessKey := "HB3hT1ACjFaWZvzc8sGj70JCWqQLhLkm"
	useSSL := false

	// Initialize minio client object.
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
}

type UploadFileInput struct {
	Filename      string
	Filesize      int64
	Base64Content string
	ContentType   string
	BucketName    string
}

type UploadFileResult struct {
	Etag       string
	Error      string
	HumanError string
}

// UploadFile godoc
// @Summary      get document
// @Description  get document by bucket and filename
// @Tags         minio
// @Accept       json
// @Produce      json
//
//	@Param			filedata	body		UploadFileInput	true	"upload file"
//
// @Success      200  {object}  UploadFileResult
// @Failure      422  {object}  UploadFileResult
// @Failure      500  {object}  UploadFileResult
// @Router       /minio/file [post]
func UploadFile(c *gin.Context) {
	data := UploadFileInput{}
	err := c.ShouldBind(&data)
	if err != nil {
		c.JSON(422, UploadFileResult{Etag: "", Error: err.Error(), HumanError: "???? ???????????? ?????????????? ????????????"})
		return
	}
	client, err := getMinio()
	if err != nil {
		c.JSON(500, UploadFileResult{Etag: "", Error: err.Error(), HumanError: "?????????????????? s3 ????????????????????"})
		return
	}
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data.Base64Content))
	info, err := client.PutObject(context.Background(), data.BucketName, data.Filename, reader, data.Filesize, minio.PutObjectOptions{ContentType: data.ContentType})
	c.JSON(200, UploadFileResult{Etag: info.ETag})
}

type DownloadFileInput struct {
	Filename   string `form:"Filename"`
	BucketName string `form:"BucketName"`
}

type DownloadFileResult struct {
	Error         string
	HumanError    string
	ContentType   string
	FileSize      int64
	Base64Content string
}

// DownloadFile godoc
// @Summary      get document
// @Description  get document by bucket and filename
// @Tags         minio
// @Accept       json
// @Produce      json
// @Param        Filename   query  string  true  "Filename"
// @Param        BucketName   query  string  true  "BucketName"
// @Success      200  {object}  DownloadFileResult
// @Failure      422  {object}  DownloadFileResult
// @Failure      500  {object}  DownloadFileResult
// @Router       /minio/file [get]
func DownloadFile(c *gin.Context) {
	var data DownloadFileInput
	err := c.ShouldBind(&data)
	if err != nil {

		c.JSON(422, DownloadFileResult{Error: err.Error(), HumanError: "???????????????? ?????????????? ????????????"})
		return
	}
	fmt.Printf("%+v\n", data)
	fmt.Printf("%+v\n", c.Request.URL.Query())
	client, err := getMinio()
	if err != nil {
		c.JSON(500, DownloadFileResult{Error: err.Error(), HumanError: "?????????????????? s3 ????????????????????"})
		return
	}
	obj, err := client.GetObject(context.Background(), data.BucketName, data.Filename, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(500, DownloadFileResult{Error: err.Error(), HumanError: "???????????? ???????????? ?????????? ?? ??????????????????"})
		return
	}
	objStat, err := obj.Stat()
	if err != nil {
		c.JSON(500, DownloadFileResult{Error: err.Error(), HumanError: "???????????? ?????????????????? ???????????? ?? ??????????"})
		return
	}
	objReader := bufio.NewReader(obj)
	objContent, err := ioutil.ReadAll(objReader)
	if err != nil {
		c.JSON(500, DownloadFileResult{Error: err.Error(), HumanError: "???????????? ???????????? ???????? ??????????"})
		return
	}
	c.JSON(200, DownloadFileResult{ContentType: objStat.ContentType, FileSize: objStat.Size,
		Base64Content: base64.StdEncoding.EncodeToString(objContent)})
}

func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// @title           Swagger Example API
// @version         1.0
// @description     This is a sample server celler server.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		pingPong := v1.Group("ping-pong")
		{
			pingPong.GET("", Ping)
		}
		minioGroup := v1.Group("minio")
		{
			minioGroup.GET("file", DownloadFile)
			minioGroup.POST("file", UploadFile)
		}
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
