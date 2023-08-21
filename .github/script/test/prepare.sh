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

mkdir /tmp/scpw-local-dir
cd /tmp/scpw-local-dir
sudo chmod -R 777 .
echo "当前 `pwd` `stat .`"

mkdir /tmp/scpw-remote-dir
cd /tmp/scpw-remote-dir
sudo chmod -R 777 .
echo "当前 `pwd` `stat .`"

sudo mkdir /tmp/no-permission-dir
cd /tmp/no-permission-dir
sudo touch np-file1
sudo mkdir np-dir1
sudo chmod -R o-rwx /tmp/no-permission-dir
echo "当前 `pwd` `stat .`"
tree -hp

sudo touch /tmp/no-permission-file
sudo chmod o-rwx /tmp/no-permission-file
stat /tmp/no-permission-file

cd /tmp
echo "当前 `pwd` `stat .`"

cd /home/runner
echo "当前 `pwd` `stat .`"