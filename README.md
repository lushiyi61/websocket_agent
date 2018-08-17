# socket_agency
实现socket服务器能够同时处理TcpSocket和WebSocket客户端

# 运行指令:
go run .\socket_agency.go 2305 2306 127.0.0.1:12306

- go run .\socket_agency.go 
- 2305 监听 WebSocket
- 2306 监听 TcpSocket
- 127.0.0.1:12306 目标socket服务器


# 文件说明
- socket_agency.go  socket代理，工作成果
- socket_client_web.go WebSocket客户端（无，请在线测试 http://www.blue-zero.com/WebSocket/
- socket_client_tcp.go TcpSocket客户端（测试文件）
- socket_server_tcp.go 测试TcpSocket服务端（收到什么就转发什么）（测试文件）


# 测试数据（他人测试结果）

机器性能：
4核 3G内存
测试时go的版本为1.5.1


1W连接， 6W+ /s 读写次数， 4核， si在45% 左右， （每秒有10个左右的连接会等到接受超过2S）
增加到1W5连接， 还是6W/s  读写次数  si有到70% 左右 (每秒40个所有的连接会等到超过2S， 偶尔会3S)


每个连接改成1S发一条消息  1Ｗ连接没有任何压力，　１Ｗ/s的读写速度， 20%左右的si
1W5 si最高的20%左右

初始最高连接在2W8左右， 所以后边比较困难进行连接

可以修改
vi /etc/sysctl.conf   

添加下面一行： 

net.ipv4.ip_local_port_range = 1024 65535

sysctl -p 

Linux默认的可用端口范围是： 32768-61000

引用
[root@PerfTestApp3 ~]# sysctl -a|grep ip_local_port_range 
net.ipv4.ip_local_port_range = 32768    61000

将默认端口修改为1024-65535

这样就会使得默认可用端口为1024-65535

替换之后再次进行测试
每个连接1S发送一条数据， 读写为6Ｗ/s   si有到70% 左右 (每秒30个所有的连接会等到超过2S）

