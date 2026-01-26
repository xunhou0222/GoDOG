# GoDOG

GoDOG (Golang Downloader Of GNSS), is a GNSS data downloader implemented in Golang

## Features

- **Cross-platform**: GoDOG can be run on any platform that supports Golang;
- **Fast**: downloading multiple files concurrently using Goroutine;
- **Pure Golang**: GoDOG is developped in pure Golang, and does not rely on any third-party software or packages such as wget, curl, etc. GoDOG itself realizes file download based on different protocols such as FTP/FTPS, HTTP and HTTPS (CDDIS only);
- **Flexible**: Users can customize download types in the JSON file, and set the corresponding download link, user name, login password and other information.
