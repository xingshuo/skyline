# Skyline
* 基于golang实现的Actor框架

## 设计初衷
* 旨在让开发者可以忽略并发安全相关处理流程(锁和goroutine)，以写单线程逻辑的方式开发游戏业务功能

## 支持功能
* TCP网络库封装
* 以service为最小并发单元的actor模型支持
* 基于service的Frame Timer && Golang Timer封装
* 基于service的异步并发 && 异步线性并发支持
* 节点(Cluster)间RPC
* 简易日志库

## Api
* 详见[skyline.go](https://github.com/xingshuo/skyline/blob/main/skyline.go#L33)
* 注意: 没有`goroutine safe`备注的接口，只能在其对应Service的执行goroutine中被调用

## Reference
* https://github.com/cloudwu/skynet
* https://github.com/topfreegames/pitaya
* https://github.com/name5566/leaf