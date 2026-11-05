# wxcloudrun-golang
[![GitHub license](https://img.shields.io/github/license/WeixinCloud/wxcloudrun-express)](https://github.com/WeixinCloud/wxcloudrun-express)
![GitHub package.json dependency version (prod)](https://img.shields.io/badge/golang-1.17.1-green)

微信云托管 golang 模版，实现简单的计数器读写接口，使用云托管 MySQL 读写、记录计数值。

![](https://qcloudimg.tencent-cloud.cn/raw/be22992d297d1b9a1a5365e606276781.png)


## 快速开始
前往 [微信云托管快速开始页面](https://developers.weixin.qq.com/miniprogram/dev/wxcloudrun/src/basic/guide.html)，选择相应语言的模板，根据引导完成部署。

## 本地调试
下载代码在本地调试，请参考[微信云托管本地调试指南](https://developers.weixin.qq.com/miniprogram/dev/wxcloudrun/src/guide/debug/)

## 实时开发
代码变动时，不需要重新构建和启动容器，即可查看变动后的效果。请参考[微信云托管实时开发指南](https://developers.weixin.qq.com/miniprogram/dev/wxcloudrun/src/guide/debug/dev.html)

## Dockerfile最佳实践
请参考[如何提高项目构建效率](https://developers.weixin.qq.com/miniprogram/dev/wxcloudrun/src/scene/build/speed.html)

## 目录结构说明
~~~
.
├── Dockerfile                Dockerfile 文件
├── LICENSE                   LICENSE 文件
├── README.md                 README 文件
├── container.config.json     模板部署「服务设置」初始化配置（二开请忽略）
├── db                        数据库逻辑目录
├── go.mod                    go.mod 文件
├── go.sum                    go.sum 文件
├── index.html                主页 html 
├── main.go                   主函数入口
└── service                   接口服务逻辑目录
~~~


## 服务 API 文档

### `GET /api/count`

获取当前计数

#### 请求参数

无

#### 响应结果

- `code`：错误码
- `data`：当前计数值

##### 响应结果示例

```json
{
  "code": 0,
  "data": 42
}
```

#### 调用示例

```
curl https://<云托管服务域名>/api/count
```



### `POST /api/count`

更新计数，自增或者清零

#### 请求参数

- `action`：`string` 类型，枚举值
  - 等于 `"inc"` 时，表示计数加一
  - 等于 `"clear"` 时，表示计数重置（清零）

##### 请求参数示例

```
{
  "action": "inc"
}
```

#### 响应结果

- `code`：错误码
- `data`：当前计数值

##### 响应结果示例

```json
{
  "code": 0,
  "data": 42
}
```

#### 调用示例

```
curl -X POST -H 'content-type: application/json' -d '{"action": "inc"}' https://<云托管服务域名>/api/count
```

## 使用注意
如果不是通过微信云托管控制台部署模板代码，而是自行复制/下载模板代码后，手动新建一个服务并部署，需要在「服务设置」中补全以下环境变量，才可正常使用，否则会引发无法连接数据库，进而导致部署失败。
- MYSQL_ADDRESS
- MYSQL_PASSWORD
- MYSQL_USERNAME
以上三个变量的值请按实际情况填写。如果使用云托管内MySQL，可以在控制台MySQL页面获取相关信息。



## 后台管理（二次开发新增）

访问 `http://<服务域名>/admin` 打开管理页面，可以查看计数、自增、设为指定值、清零、新建与删除记录，并显示数据库延迟与服务运行时长。

### 接口一览

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/admin/overview` | 概览：当前计数、记录数、数据库延迟、运行时长 |
| GET | `/api/admin/counters` | 全部计数记录 |
| POST | `/api/admin/counters` | 新建记录，body `{"count":0}`，可省略 body |
| POST | `/api/admin/counters/incr` | 自增，body `{"id":1,"step":1}` |
| POST | `/api/admin/counters/set` | 设为固定值，body `{"id":1,"count":10}` |
| POST | `/api/admin/counters/delete` | 删除记录，body `{"id":2}` |

### 访问控制

在云托管「服务设置 → 环境变量」中配置 `ADMIN_TOKEN` 后，所有管理接口都需要携带请求头
`X-Admin-Token: <你的密钥>`（或查询参数 `?token=`）才会放行，未携带返回 401；
页面首次访问会弹窗要求输入密钥，密钥保存在浏览器本地。

**未配置 `ADMIN_TOKEN` 时管理页面完全公开**，公网环境请务必配置，否则任何人都能改数据。

### 本地调试

```bash
export MYSQL_USERNAME=xxx MYSQL_PASSWORD=xxx
export MYSQL_ADDRESS=10.0.0.0:3306 MYSQL_DATABASE=golang_demo
go run .              # 默认监听 80 端口
PORT=8080 go run .    # 或指定端口
```

本地不在云托管 VPC 内时，`MYSQL_ADDRESS` 要填 MySQL 的**外网地址**。



## License

[MIT](./LICENSE)
