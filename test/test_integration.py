#!/usr/bin/python3

import json
import os
import pathlib
import random
import string
import subprocess
import tempfile
from urllib.parse import urlsplit

import pytest

from vm import VM


def img_cache_path():
    return pathlib.Path(os.getenv("TEST_IMG_CACHE", "./cache"))


@pytest.fixture(name="fedora_vm")
def fedora_vm_fixture():
    # XXX: shold this use centos stream instead?
    # XXX2: is there a stable download url?
    # XXX3: should we just generate our own image :) ?
    img_url = "https://download.fedoraproject.org/pub/fedora/linux/releases"\
        "/39/Cloud/x86_64/images/Fedora-Cloud-Base-39-1.5.x86_64.qcow2"

    cache_path = img_cache_path()
    cache_path.mkdir(parents=True, exist_ok=True)
    img_final_path = pathlib.Path(f"{cache_path}/{os.path.basename(urlsplit(img_url).path)}")
    if img_final_path.exists():
        return VM(img_final_path)

    # TODO: make this a helper
    # image missing
    img_download_path = pathlib.Path(img_final_path).with_suffix(".downloaded")
    subprocess.check_call([
        "curl", "-L", "-C", "-", "-o", os.fspath(img_download_path), img_url])
    # TODO: make part of VM
    # TODO2: sad, that this is resize is not happening automatically
    img_resized_path = img_download_path.with_suffix(".resized")
    subprocess.check_call([
        "qemu-img", "create", os.fspath(img_resized_path), "20G"])
    subprocess.check_call([
        "virt-resize",
        "--expand", "/dev/sda5",
        os.fspath(img_download_path), os.fspath(img_resized_path)])
    os.remove(os.fspath(img_download_path))
    # XXX: make this nicer
    vm = VM(img_resized_path)
    # allow root login
    vm.enable_root_ssh()
    # and put in place
    os.rename(os.fspath(img_resized_path), os.fspath(img_final_path))
    return VM(img_final_path)


@pytest.fixture(name="fedora_vm_with_podman")
def fedora_vm_with_podman_fixture(fedora_vm):
    fedora_vm_podman_path = pathlib.Path(os.fspath(fedora_vm._img) + ".customized")
    if fedora_vm_podman_path.exists():
        return VM(fedora_vm_podman_path)
    # no cached VM, create new
    wip = fedora_vm_podman_path.with_suffix(".wip")
    # TODO: make this a proper "VM.clone_with()" call or something
    #       (and use snapshots?!?)
    subprocess.check_call(
        ["cp", "-a", fedora_vm._img, wip])
    subprocess.check_call(
        ["cp", "-a", fedora_vm._img.with_suffix(".password"), wip.with_suffix(".password")])
    with VM(wip, snapshot=False) as vm:
        exit_status, output = vm.run("dnf install -y podman")
        assert exit_status == 0, f"failed with {output}"
        # TODO: is this needed once we build locally?
        # pull the container already too
        exit_status, output = vm.run("podman pull ghcr.io/osbuild/osbuild-deploy-container")
        assert exit_status == 0, f"failed with {output}"
    wip.rename(fedora_vm_podman_path)
    return VM(fedora_vm_podman_path)


def send_local_tree(vm):
    source_path = pathlib.Path(__file__).parent.parent
    with tempfile.TemporaryDirectory() as tmpd:
        tree_tar_path = pathlib.Path(tmpd) / "tree.tar.gz"
        subprocess.check_call([
            "tar", "-c", "-z",
            # TODO: exclude .gitexclude
            "--exclude=cache", "--exclude=.git",
            "-f", os.fspath(tree_tar_path),
            "-C", os.fspath(source_path),
            ".",
        ])
        # put into vm
        vm.start()
        vm.run("rm -rf /tests")
        vm.run("mkdir /tests")
        vm.put(tree_tar_path, "/tmp/tree.tar.gz")
        vm.run("tar -x -C /tests -f /tmp/tree.tar.gz")


# TODO: test less in one go somehow?
def test_osbuild_deploy_container_full(fedora_vm_with_podman):
    send_local_tree(fedora_vm_with_podman)

    # pass config.json to have root login
    pw = ''.join(random.choices(string.hexdigits, k=12))
    print(f"generated pw for the test vm {pw}")
    # TODO: how to setup root pw?
    blp = json.dumps({
        "blueprint": {
            "customizations": {
                "user": [
                    {
                        "name": "test",
                        "password": pw,
                        "groups": ["wheel"],
                    },
                ],
            },
        },
    })
    exit_status, output = fedora_vm_with_podman.run("mkdir -v output")
    assert exit_status == 0, f"failed with {output}"
    fedora_vm_with_podman.run(f"echo '{blp}' > output/config.json")
    assert exit_status == 0, f"failed with {output}"
    # TODO: remove, debug only
    # fedora_vm_with_podman.get("output/config.json", "/tmp/config.json")

    # build local container from source
    exit_status, output = fedora_vm_with_podman.run(
        "podman build -f /tests/Containerfile -t osbuild-deploy-container-test")
    assert exit_status == 0, f"failed with {output}"

    exit_status, output = fedora_vm_with_podman.run(
        "podman run --rm --privileged --security-opt label=type:unconfined_t "
        "-v $(pwd)/output:/output "
        "osbuild-deploy-container-test quay.io/centos-boot/fedora-tier-1:eln "
        "--config /output/config.json")
    assert exit_status == 0, f"failed with {output}"

    exit_status, output = fedora_vm_with_podman.run("ls -lR output")
    assert exit_status == 0, f"failed with {output}"

    # TODO: remove
    # DEBUG only
    # fedora_vm_with_podman.get("output/qcow2/disk.qcow2", "/tmp/disk.qcow2")

    exit_status, output = fedora_vm_with_podman.run("journalctl | grep osbuild | grep denied")
    assert exit_status == 1, f"found selinux denials {output}"

    print(f"using pw for the test vm {pw}")
    # get the image and test it
    with tempfile.TemporaryDirectory() as tmpd:
        generated_disk_path = pathlib.Path(tmpd) / "disk.qcow2"
        fedora_vm_with_podman.get("output/qcow2/disk.qcow2", generated_disk_path)
        with VM(generated_disk_path, user="test", password=pw) as test_vm:
            test_vm.start()
            # TODO: user creation in osbuild-deploy-container is not ready yet
            # test_vm.run("true")
            # test_vm.run("id")
            ready = test_vm._wait_ssh_ready()
            assert ready
            # cannot root login right now so force stop
            test_vm.force_stop()
