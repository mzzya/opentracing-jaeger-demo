### go tracing demo
观察http下traing原理 header部分

##### 第一步 启动jaeger
##### 第二步 启动抓包代理 打开 http://127.0.0.1:8899/#network
##### 第三步 go run main.go
##### 第四步 打开 http://localhost:16686/search

### 结论
http接口调用时是在 header中注入了
Uber-Trace-Id 1f3c1b1a0802da73:1f3c1b1a0802da73:0:1


### 参考资料

https://opentracing.io/ opentracing首页 docker 启动 Jaeger

https://wu-sheng.gitbooks.io/opentracing-io/content/pages/quick-start.html opentracing中文介绍 <br>
根据大佬demo 输出到jaeger

http://wproxy.org/whistle/ node+web的抓包工具


### jaeger相关
官方文档 https://www.jaegertracing.io/docs/1.11/<br>
部署方式 https://github.com/jaegertracing/jaeger-kubernetes<br>
