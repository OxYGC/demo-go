package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/gin-gonic/gin"
)

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func main1() {
	r := gin.Default()

	// 静态资源
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, htmlIndex)
	})

	// 读取文件
	r.GET("/api/file", func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.JSON(400, gin.H{"error": "缺少参数 path"})
			return
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"content": string(data)})
	})

	// 保存文件
	r.POST("/api/file", func(c *gin.Context) {
		var req struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if req.Path == "" {
			c.JSON(400, gin.H{"error": "缺少参数 path"})
			return
		}
		err := ioutil.WriteFile(req.Path, []byte(req.Content), 0644)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "保存成功"})
	})

	port := 8080
	url := fmt.Sprintf("http://localhost:%d", port)
	go func() {
		_ = openBrowser(url)
	}()

	fmt.Println("服务已启动，打开浏览器访问：", url)
	err := r.Run(fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("服务启动失败:", err)
	}
}

const htmlIndex = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <title>本地文件编辑器</title>
  </head>
  <body>
    <h2>本地文件编辑器</h2>
    <input id="path" style="width:350px" placeholder="请输入本地文件路径">
    <button onclick="loadFile()">读取文件</button>
    <br><br>
    <textarea id="content" style="width:98%;height:400px"></textarea>
    <br>
    <button onclick="saveFile()">保存文件</button>
    <span id="status"></span>
    <script>
      function loadFile() {
        let path = document.getElementById('path').value;
        fetch('/api/file?path=' + encodeURIComponent(path))
        .then(r => r.json())
        .then(res => {
          if(res.error){
            alert(res.error)
          }else{
            document.getElementById('content').value = res.content
            document.getElementById('status').textContent = ""
          }
        })
      }
      function saveFile() {
        let path = document.getElementById('path').value;
        let content = document.getElementById('content').value;
        fetch('/api/file', {
          method: 'POST',
          headers: {'Content-Type':'application/json'},
          body: JSON.stringify({path: path, content: content})
        })
        .then(r => r.json())
        .then(res => {
          if(res.error){
            alert(res.error)
          }else{
            document.getElementById('status').textContent = res.message
          }
        })
      }
    </script>
  </body>
</html>
`
