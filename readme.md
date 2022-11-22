> 运行环境

* 需要安装谷歌浏览器

> 如何运行

* 安装go的运行环境
* window环境下，可直接双击main.exe
* 也可以命令行启动，输入: 

```
cd gochromedp
main.exe
```

 


> 需要在config.xlsx文件里输入账号密码

* 第一列为账号
* 第二列为密码
* 页签名为：Sheet1

> 代理配置iplist.txt，每行一个代理，格式 如下两种
* 127.0.0.1:8888
* http://127.0.0.1:8888


> 运行结果：
 
* 在运行目录下会输出 Result_yyyyMMdd.txt文件

> 多线程工作
* 5个工作线程，循环工作
* chan输出结果
 
