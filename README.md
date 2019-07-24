# kisscluster

[![MIT licensed][1]][2]
[![Go Report Card][3]][4]

[1]: https://img.shields.io/badge/license-MIT-blue.svg
[2]: LICENSE.md
[3]: https://goreportcard.com/badge/github.com/nothollyhigh/kisscluster
[4]: https://goreportcard.com/report/github.com/nothollyhigh/kisscluster


- 本项目为 KISS 组件包集群服务器示例代码，每个游戏项目需求不同，具体集群请根据自家业务实现

## 目录结构

### 0. kisscluster/conf

- 配置文件目录，配置文件说明详见具体配置文件和代码

### 1. kisscluster/gate

- 网关服务器，简单反代websocket/tcp线路到后端节点，默认配置了websocket反代到plaza、game的tcp线路

### 2. kisscluster/center

- 中心服务器，负责集群管理，接收大厅服务器、游戏服务器注册，并更新游戏服务器列表给大厅服务器

### 3. kisscluster/plaza

- 大厅服务器，注册到中心服务器，接受客户端登录请求，接收中心服务器更新游戏服务器列表并同步游戏服务器列表给客户端

### 4. kisscluster/game

- 游戏服务器，注册到中心服务器，暂未加具体的游戏逻辑

### 5. kisscluster/robot

- 示范的机器人代码，通过网关websocket协议登录到大厅服务器并接收游戏服务器列表


## 构建

```sh
cd kisscluster

go mod init kisscluster

go build center/center.go
go build plaza/plaza.go
go build game/game.go
go build gate/gate.go
go build robot/robot.go
```

## 运行

- 启动各服务器节点和机器人，观察日志

1. 启动 center

2. 启动 plaza

3. 启动 game

4. 启动 gate

5. 启动 robot
