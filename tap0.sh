#!/bin/bash
ip link set tap0 up mtu 1400
ip addr add 10.0.0.1/24 dev tap0
ip -6 addr add 2001::1/24 dev tap0
ip link set dev tap0 up
