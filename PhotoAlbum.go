package main

import (
	"fmt"                // 导入fmt包，用于格式化输入输出
	"photoAlbum/config"  // 导入photoAlbum项目的config包
	"photoAlbum/global"  // 导入photoAlbum项目的global包
	"photoAlbum/service" // 导入photoAlbum项目的service包
	"photoAlbum/updata"
	"strconv" // 导入strconv包，用于字符串转换

	"github.com/gofiber/fiber/v2"         // 导入gofiber框架的v2版本
	"github.com/gofiber/template/html/v2" // 导入gofiber框架的html模板引擎的v2版本
)

func main() {

	// 定义一个错误变量
	var err error
	// 从config.yaml文件中读取配置信息，并赋值给global.Config
	global.Config, err = config.New("./config.yaml")
	// 如果读取配置信息出错，则打印错误信息并返回
	if err != nil {
		fmt.Println(err)
		return
	}
	// 初始化相册列表，并赋值给global.PhotoAlbumList
	global.PhotoAlbumList, err = service.InitPhotoAlbum(global.Config.PhotoAlbumAbsolutePath)

	// 如果初始化相册列表出错，则打印错误信息并返回
	if err != nil {
		fmt.Println(err)
		return
	}

	updata.StartPhotoAlbumUpdater()

	// 创建一个html引擎，用于渲染模板
	engine := html.New("./views", ".html")
	// 创建一个fiber应用，并配置视图引擎
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// 设置静态文件目录
	app.Static("/public", "./public")

	// 定义一个路由，用于处理请求
	app.Get("/:pageNum?/:pageSize?", func(c *fiber.Ctx) error {

		// 从请求参数中获取pageNum和pageSize，并转换为整数
		pageNum, err := strconv.Atoi(c.Params("pageNum"))
		// 如果pageNum转换出错，则默认为1
		if err != nil {
			pageNum = 1
		}
		pageSize, err := strconv.Atoi(c.Params("pageSize"))
		// 如果pageSize转换出错，则默认为2
		if err != nil {
			pageSize = 2
		}

		// 调用PhotoAlbumList的Pagination方法，获取分页后的相册列表
		photoAlbumList, p := global.PhotoAlbumList.Pagination(pageNum, pageSize)
		// 渲染index模板，并传递Config、PhotoAlbumList和Pagination参数
		return c.Render("index", fiber.Map{
			"Config":         global.Config,
			"PhotoAlbumList": photoAlbumList,
			"Pagination":     p,
		})
	})

	// 启动应用，监听global.Config.ListenPort端口
	err = app.Listen(":" + global.Config.ListenPort)

	// 如果启动应用出错，则打印错误信息并返回
	if err != nil {
		fmt.Println(err)
		return
	}

}
