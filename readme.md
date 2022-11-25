> ./gorun.bat 执行go run 

> ./start.bat 执行main.exe 2  ,2代表执行两次任务切换一个IP

> ./deploy.bat 生成32位的可执行文件 

> 运行环境

* 需要安装谷歌浏览器
* 只能在windows下运行，因为使用了adsl拨号来切换IP

> 如何运行

* 安装go的运行环境
* window环境下，可直接双击main.exe
* 也可以命令行启动，输入: 

```
cd gochromedp
go run main.go adsl.go
```

 


> 需要在config.xlsx文件里输入账号密码

* 第一列为账号
* 第二列为密码
* 页签名为：Sheet1

> 代理配置iplist.txt，每行一个代理，格式 如下两种
* 127.0.0.1:8888
* http://127.0.0.1:8888

> adsl账号配置adsl_config.txt，格式：
* 第一行为adsl账号
* 第二行为adsl密码

> 运行结果输出： 
* 在运行目录下会输出 Result_yyyyMMdd.txt文件


> **目前使用adsl拨号，使用单线程工作**

> ~~多线程工作~~
* 5个工作线程，循环工作
* chan输出结果
 
> windows测试：双击start.bat文件即可，
 默认执行两次任务切换一次IP，可自行修改start.bat文件中的数字，代表执行几次任务切换一个IP