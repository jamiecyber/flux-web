package controllers

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"flux-web/models"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/httplib"
	"github.com/astaxie/beego/logs"
)

type WorkloadController struct {
	beego.Controller
}


var l = logs.GetLogger()

var flux = models.Flux{
	FluxUrl:            os.Getenv("FLUX_URL"),
	SyncApi:            "/api/flux/v6/sync?ref=",
	JobApi:             "/api/flux/v6/jobs?id=",
	UpdateManifestsApi: "/api/flux/v9/update-manifests",
	ListImagesApi:      "/api/flux/v10/images?namespace=", 
}

func (this *WorkloadController) ListWorkloads() {
	ns := this.Ctx.Input.Param(":ns")
	l.Printf("in ListWorkloads, executing: " + flux.FluxUrl + flux.ListImagesApi + ns)
	res, err := httplib.Get(flux.FluxUrl + flux.ListImagesApi + ns).Debug(true).Bytes()
	if err != nil {
		l.Panic(err.Error)
	}
	this.Ctx.Output.Body(res)
}

func (this *WorkloadController) ReleaseWorkloads() {
	jobID, err := triggerJob(this.Ctx.Input.RequestBody)
	if err != nil {
		l.Printf(err.Error())
		this.Ctx.Output.SetStatus(500)
		return
	}
	this.Ctx.WriteString("Done")

	// @Refactor: Here we should implement ws response when release was done.
	newReleaseRequest, err := models.NewReleseRequest(this.Ctx.Input.RequestBody)
	if err != nil {
		l.Printf(err.Error())
		return
	}
	go waitForSync(jobID, newReleaseRequest)
}

func waitForSync(jobID string, releaseRequest models.ReleaseRequest) {

	syncID, err := getSyncID(jobID)
	if err != nil {
		l.Printf(err.Error())
		return
	}

	for {
		resp, err := httplib.Get(flux.FluxUrl + flux.SyncApi + syncID).String()
		if err != nil {
			l.Printf(err.Error())
			break
		}
		if resp == "[]" {
			l.Printf("release for" + releaseRequest.Spec.ContainerSpecs.Workload[0].Container + " is done!")
			releaseChannel <- releaseRequest.Spec.ContainerSpecs.Workload[0].Container + " done!"
			break
		}
		time.Sleep(time.Millisecond * 300)
	}
}

func getSyncID(jobID string) (string, error) {
	l.Printf("getting syncID...")
	
	for {
		resp, err := httplib.Get(flux.FluxUrl + flux.JobApi + jobID).Bytes()
		if err != nil {
			l.Printf(err.Error())
			return "", errors.New(err.Error())
		}
		job, err := models.NewJob(resp)
		if err != nil {
			l.Panic("Error_getSyncID_01: " + err.Error())
			return "", errors.New(err.Error())
		}
		if job.Result.Revision != "" {
			l.Printf("got syncID: " + job.Result.Revision)
			return job.Result.Revision, nil
		} else if job.Err != "" {
			l.Printf("Error_getSyncID_02")
			releaseChannel <- "Error_getSyncID_02"
			return job.Err, errors.New(job.Err)
		} else {
			l.Printf("job status: " + job.StatusString)
		}
		time.Sleep(time.Second)
	}
	return "", nil
}

func triggerJob(requestBody []byte) (string, error) {
	resp, err := http.Post(flux.FluxUrl+flux.UpdateManifestsApi, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		l.Printf("Error_triggerJob_01: " + err.Error())
		return "", errors.New(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			l.Panic(err.Error)
			return "", errors.New(err.Error())
		}
		l.Printf(string(bodyBytes))
		jobID := strings.Replace(string(bodyBytes), "\"", "", -1)
		l.Printf("job " + jobID + " triggered")
		return string(jobID), nil
	} else {
		return "", errors.New("Job request statuscode is: " + string(resp.StatusCode))
	}
}

func GetImages(params ...string) []models.Image {
	namespace := os.Getenv("DEFAULT_NAMESPACE")
	if len(params) > 0 {
		namespace = params[0]
		l.Printf(namespace)
	}
	res, err := httplib.Get(flux.FluxUrl + flux.ListImagesApi + namespace).Debug(true).Bytes()
	if err != nil {
		l.Panic(err.Error)
	}

	images, err := models.NewImages(res)
	if err != nil {
		l.Panic(err.Error)
	}
	if len(params) > 1 {
		filter := params[1]
		l.Printf(filter)
		for i := 0; i < len(images); i++ {
			if !strings.Contains(images[i].ID, filter) {
				images = append(images[:i], images[i+1:]...)
				i--
			}
		}
	}
	return images
}