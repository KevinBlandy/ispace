
# iSpace

个人云盘系统，不依赖任何第三方服务，使用 SQLite 作为数据库。

* 用户系统：`http(s)//:<host>/`
* 管理系统：`http(s)//:<host>/#/manager`

管理系统默认的用户名和密码均是 `root`（首次登录后千万记得修改）。

## 系统设计

* 物理文件和逻辑文件分离，通过 hash 实现秒传。
* 符合条件的文件存储使用 Gzip 压缩，节省空间。
* 通过断点续传实现大文件上传。
* 支持异步下载远程资源，并且通过 SSE 监控下载进度。
* 支持 Range 协商，客户端实现断点下载。
* 支持文件分享。
* 前后端分离，但通过 Embed 嵌入进整个客户端，不需要单独部署前端项目。
* 独立的可执行文件，不依赖任何第三方的服务，使用 SQLite 数据库，JWT，部署即可用。

## 启动命令

```shell
./ispace \
  --db "database/db" \
  --log.dir "logs" \
  --public.dir "public" \
  --store.dir "storage" \
  --chunk.dir "chunk" \
  --http.port "8689" \
  --http.host "0.0.0.0" \

```

参数如下：

* `db` SQL 数据库文件
* `log.dir`  指定日志输出目录
* `public.dir` 公共资源目录，这里面的资源可以被直接访问
* `store.dir` 存储上传文件的目录
* `chunk.dir` 存储分片上传的临时文件的目录
* `http.port` http 服务端口
* `http.host` http 服务主机

上述配置都有默认值，即示例中的参数。

## 网关相关的一些配置

如果使用了 nginx 之类的网关，有一些需要注意的地方。

```nginx
# 关掉 nginx 对请求体的缓存
# 要客户端数据实时流式转发给后端，后端才能边接收边写盘，断点续传接口中的 received 才会正确递增。
proxy_request_buffering off;

# 不限制客户端请求体大小
client_max_body_size 0;
```


