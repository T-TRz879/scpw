#!/script/bash

# 更新包
sudo apt install -y openssh-client openssh-server tree
sudo apt update

# 启动ssh服务
sudo /etc/init.d/ssh start
ssh_pid=$(ps -e | grep sshd | awk '{print $1}')

# 检查ssh_pid是否为空
if [ -n "$ssh_pid" ]; then
    # 如果存在ssh进程，输出对应的pid
    echo "SSH进程的PID为: $ssh_pid"
else
    # 如果不存在ssh进程，输出提示信息
    echo "无SSH进程"
fi

sudo useradd -m -s /bin/bash -p $(openssl passwd -1 "scpwuser123") scpwuser

cd /tmp
mkdir scpw-test-dir
cd scpw-test-dir
mkdir dir1
echo "12345" > file1
mkdir dir1/dir1_1
echo "12345" > dir1/file1_1
echo "12345" > dir1/file1_2
mkdir dir1/dir1_2

echo "用户和工作目录创建完成！"

# 到 /home/test1 和 /home/test2 目录下
cd /tmp/scpw-test-dir
stat .
pwd
tree -hp