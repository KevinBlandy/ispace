
# iSpace

个人云盘系统，不依赖任何第三方服务，使用 SQLite 作为数据库。

* 用户系统：`http(s)//:<host>/`
* 管理系统：`http(s)//:<host>/#/manager`

## 启动命令

```shell
./ispace \
  --db "database/db" \
  --log.dir "logs" \
  --public.dir "public" \
  --store.dir "storage" \
  --http.port "8689" \
  --http.host "0.0.0.0" \

```

* `db` SQL 数据库文件
* `log.dir`  指定日志输出目录
* `public.dir` 公共资源目录，这里面的资源可以被直接访问
* `store.dir` 存储上传文件的目录
* `http.port` http 服务端口
* `http.host` http 服务主机

上述配置都有默认值，即示例中的参数。


## Build

```shell
CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ GOARCH=amd64 GOOS=linux CGO_ENABLED=1 go build -ldflags "-linkmode external -extldflags -static"
```
