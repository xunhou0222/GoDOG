# GoDOG

GoDOG (Golang Downloader Of GNSS)，是一款用Go语言开发的GNSS数据下载软件

## 特点

- **跨平台**：任何支持Go语言的系统均支持GoDOG；
- **快速**：Goroutine并发下载多个文件；
- **纯粹**：纯粹Go语言开发，不依赖wget、curl等任何第三方软件和第三方包，自实现基于FTP/FTPS、HTTP与HTTPS（仅限于CDDIS）等不同协议的文件下载；
- **易拓展**：用户可在json文件中自定义下载类型，设置相应的下载链接、用户名和登录密码等信息；
