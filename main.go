package main

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var templatesFS embed.FS

func main() {
	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	// 使用 embed 加载模板
	templ := template.Must(template.New("").ParseFS(templatesFS, "templates/*"))
	r.SetHTMLTemplate(templ)

	// 设置静态文件路径
	r.Static("/static", "./static")

	// 首页路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 代理路由
	r.Any("/proxy/*url", func(c *gin.Context) {
		// 获取URL并补充冒号
		targetURL := c.Param("url")[1:] // 移除开头的斜杠
		if strings.HasPrefix(targetURL, "https") {
			targetURL = strings.Replace(targetURL, "https", "https:", 1)
		}

		// 创建代理请求
		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "创建请求失败")
			return
		}

		// 复制原始请求的 header
		for key, values := range c.Request.Header {
			if key != "Host" { // 跳过 Host header
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}
		}

		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.String(http.StatusBadGateway, "代理请求失败")
			return
		}
		defer resp.Body.Close()

		// 从URL中获取文件名
		fileName := path.Base(targetURL)
		// 如果URL以斜杠结尾，使用默认文件名
		if fileName == "/" || fileName == "." {
			fileName = "downloaded_file"
		}

		// 设置 Content-Disposition: attachment 并包含文件名
		c.Header("Content-Disposition", `attachment; filename="`+fileName+`"`)

		// 复制响应 header
		for key, values := range resp.Header {
			for _, value := range values {
				// 跳过原始响应中的 Content-Disposition，使用我们设置的
				if key != "Content-Disposition" {
					c.Header(key, value)
				}
			}
		}

		// 设置响应状态码
		c.Status(resp.StatusCode)

		// 复制响应体
		io.Copy(c.Writer, resp.Body)
	})

	r.Run(":4080")
}
