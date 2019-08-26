package main

import (
	"flux-web/controllers"

	"github.com/astaxie/beego"
)

func main() {

	apiNs := beego.NewNamespace("/api",
		beego.NSNamespace("/v1",
			beego.NSRouter("/namespaces", &controllers.NamespaceController{}, "get:ListNamespaces"),
			beego.NSRouter("/workloads/:ns", &controllers.WorkloadController{}, "get:ListWorkloads"),
			beego.NSRouter("/release", &controllers.WorkloadController{}, "post:ReleaseWorkloads"),
		),
	)
	beego.AddNamespace(apiNs)

	beego.Router("/ws/v1", &controllers.WebSocketController{}, "get:ReleaseWorkloads")

	beego.Run()
}

