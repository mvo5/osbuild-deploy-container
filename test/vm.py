import os
import pathlib
import random
import string
import subprocess
import sys
import socket
import time
from io import StringIO

from paramiko.client import AutoAddPolicy, SSHClient


# Todo:
# - support loadvm/savevm via the monitor fd
class VM:
    MEM = "4000"

    def __init__(self, img, snapshot=True, user="root", password=""):
        self._img = img
        self._qemu_p = None
        self._snapshot = snapshot
        # there is no race free way to get a free port and pass to qemu via CLI
        self._port = 10022 + random.randint(1, 1000)
        self._user = user
        self._password = password

    def __del__(self):
        self.force_stop()

    def start(self):
        qemu_cmdline = [
            "qemu-system-x86_64", "-enable-kvm",
            "-m", self.MEM,
            # get "illegal instruction" inside the VM otherwise
            "-cpu", "host",
            # use file:/tmp/{tmpdir}serial.log here
            "-serial", f"file:{self._img_log_path()}",
            "-netdev", f"user,id=net.0,hostfwd=tcp::{self._port}-:22",
            "-device", "rtl8139,netdev=net.0",
        ]
        if self._snapshot:
            qemu_cmdline.append("-snapshot")
        qemu_cmdline.append(self._img)

        # XXX: use systemd-run to ensure cleanup?
        self._qemu_p = subprocess.Popen(
            qemu_cmdline, stdout=sys.stdout, stderr=sys.stderr)
        # XXX: also check that qemu is working and did not crash
        self._wait_ssh_ready()
        self._log(f"vm ready at port {self._port}")

    def _wait_ssh_ready(self):
        max_wait = 120
        sleep = 5
        for i in range(max_wait // sleep):
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                try:
                    s.connect(("localhost", self._port))
                    data = s.recv(256)
                    if b"OpenSSH" in data:
                        return True
                except ConnectionRefusedError:
                    time.sleep(sleep)
        # XXX: raise Exception?
        return False

    def _img_password_path(self):
        return pathlib.Path(self._img).with_suffix(".password")

    def _img_log_path(self):
        return pathlib.Path(self._img).with_suffix(".log")

    def _pw(self):
        if self._password:
            return self._password
        return self._img_password_path().read_text(encoding="utf-8")

    def force_stop(self):
        if self._qemu_p:
            self._qemu_p.kill()
            self._qemu_p = None

    def shutdown(self):
        if self._qemu_p:
            self.run("shutdown -h now")
            self._qemu_p.wait()
            self._qemu_p = None

    def _log(self, msg):
        # todo: use logger
        sys.stdout.write(msg.rstrip("\n") + "\n")
        sys.stdout.flush()

    def run(self, cmd):
        if not self._qemu_p:
            self.start()
        self._log(f"Running {cmd}")
        # TODO: make context manager
        client = SSHClient()
        client.set_missing_host_key_policy(AutoAddPolicy)
        client.connect(
            "localhost", self._port, self._user, self._pw(),
            allow_agent=False, look_for_keys=False)
        chan = client.get_transport().open_session()
        chan.get_pty()
        chan.exec_command(cmd)
        stdout_f = chan.makefile()
        # TODO: support feeding data in
        # stdin_f = chan.makefile_stdin()
        output = StringIO()
        while True:
            out = stdout_f.readline()
            if not out:
                break
            self._log(out)
            output.write(out)
        exit_status = stdout_f.channel.recv_exit_status()
        return exit_status, output.getvalue()

    def __enter__(self):
        self.start()
        return self

    def __exit__(self, type, value, tb):
        self.shutdown()

    def enable_root_ssh(self):
        if self._qemu_p:
            raise Exception("cannot use enable_root_ssh when vm is running")
        # customize, not using cloud-init to be more flexible
        img_password_path = self._img_password_path()
        if not img_password_path.exists():
            generated_pw = ''.join(random.choices(string.hexdigits, k=14))
            img_password_path.write_text(generated_pw, encoding="utf-8")
        password = img_password_path.read_text(encoding="utf-8")
        subprocess.check_call(
            ["virt-customize", "-a", os.fspath(self._img),
             "--root-password", f"password:{password}",
             "--append-line", "/etc/ssh/sshd_config:PermitRootLogin yes",
             ])
        # boot the vm once to ensure relabeling etc is done
        with VM(self._img, snapshot=False) as vm:
            vm.run("true")
        # TODO: support adding custom things like podman

    def put(self, local_path, remote_path):
        # make context manager
        client = SSHClient()
        client.set_missing_host_key_policy(AutoAddPolicy)
        client.connect(
            "localhost", self._port, self._user, self._pw(),
            allow_agent=False, look_for_keys=False)
        sftp = client.open_sftp()
        sftp.put(local_path, remote_path)
        sftp.close()

    def get(self, remote_path, local_path):
        client = SSHClient()
        client.set_missing_host_key_policy(AutoAddPolicy)
        client.connect(
            "localhost", self._port, self._user, self._pw(),
            allow_agent=False, look_for_keys=False)
        sftp = client.open_sftp()
        sftp.get(remote_path, local_path)
        sftp.close()
