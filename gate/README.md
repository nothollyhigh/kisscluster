## kiss gate

[![MIT licensed][1]][2]
[![Go Report Card][3]][4]

[1]: https://img.shields.io/badge/license-MIT-blue.svg
[2]: LICENSE.md
[3]: https://goreportcard.com/badge/github.com/nothollyhigh/kissgate
[4]: https://goreportcard.com/report/github.com/nothollyhigh/kissgate


- kiss/net 网关，反代 tcp、websocket 协议到后端 tcp 线路，

- 支持线路检测、负载均衡、realip等，详见源码


## 安装

- go get github.com/nothollyhigh/kissgate


## 运行

- kissgate -config="config.xml"


## 配置

```xml
<setting>
    <!-- debug: 设置日志是否输出到控制台 -->
    <!-- logdir: 日志目录 -->
    <!-- redirect: 是否开启全局tcp重定向 -->
    <options debug="true" logdir="./logs/" redirect="true">
        <heartbeat interval="60" timeout="50"/>
    </options>

    <proxy>
        <!-- tcp 10000 端口 反代到 tcp 10001 10002 端口 -->
        <line name="tcp" addr=":10000" type="tcp">
            <node addr="127.0.0.1:10001" maxload="50000"/>
            <node addr="127.0.0.1:10002" maxload="50000"/>
        </line>

        <!-- websocket 20000 端口, 路由 /gate/ws, 反代到 tcp 20001 20002 端口 -->
        <line name="ws" addr=":20000" type="websocket" tls="false">
            <route path="/gate/ws"/>
            <node addr="127.0.0.1:20001" maxload="50000"/>
            <node addr="127.0.0.1:20002" maxload="50000"/>
        </line>
    </proxy>
</setting>
```


## 示例

-  使用上面示例的配置启动网关

```sh
kissgate -config="config.xml"
```

- 启动后端tcp服务器，tcp/websocket各两个端口

```
cd kissgate/example
go run tcpserver.go
```

- 启动测试用客户端

```
cd kissgate/example
go run gateclient.go
```

- 观察网关、客户端、服务器日志，代码详见

1. [server](https://github.com/nothollyhigh/kissgate/blob/master/example/tcpserver.go)

2. [client](https://github.com/nothollyhigh/kissgate/blob/master/example/gateclient.go)