resource "zedcloud_model" "VM_WITH_MANY_PORTS" {
  attr = {
    "Cpus"    = "4"
    "memory"  = "8192M"
    "storage" = "100G"
  }
  brand_id       = zedcloud_brand.QEMU_VM.id
  name           = "VM_WITH_MANY_PORTS"
  origin_type    = "ORIGIN_LOCAL"
  product_status = "production"
  state          = "SYS_MODEL_STATE_ACTIVE"
  title          = "VM_WITH_MANY_PORTS"
  type           = "AMD64"

  io_member_list {
    assigngrp    = "COM1"
    cbattr       = {}
    cost         = 0
    logicallabel = "COM1"
    phyaddrs = {
      Serial = "/dev/ttyS0"
    }
    phylabel     = "COM1"
    usage        = "ADAPTER_USAGE_UNSPECIFIED"
    usage_policy = {}
    ztype        = "IO_TYPE_COM"
  }
  io_member_list {
    assigngrp    = "COM2"
    cbattr       = {}
    cost         = 0
    logicallabel = "COM2"
    phyaddrs = {
      Serial = "/dev/ttyS1"
    }
    phylabel     = "COM2"
    usage        = "ADAPTER_USAGE_UNSPECIFIED"
    usage_policy = {}
    ztype        = "IO_TYPE_COM"
  }
  io_member_list {
    assigngrp    = "COM3"
    cbattr       = {}
    cost         = 0
    logicallabel = "COM3"
    phyaddrs = {
      Serial = "/dev/ttyS2"
    }
    phylabel     = "COM3"
    usage        = "ADAPTER_USAGE_UNSPECIFIED"
    usage_policy = {}
    ztype        = "IO_TYPE_COM"
  }
  io_member_list {
    assigngrp    = "COM4"
    cbattr       = {}
    cost         = 0
    logicallabel = "COM4"
    phyaddrs = {
      Serial = "/dev/ttyS3"
    }
    phylabel     = "COM4"
    usage        = "ADAPTER_USAGE_UNSPECIFIED"
    usage_policy = {}
    ztype        = "IO_TYPE_COM"
  }
  io_member_list {
    assigngrp    = "eth0"
    cbattr       = {}
    cost         = 0
    logicallabel = "first_physical_intf"
    phyaddrs = {
      Ifname = "eth0"
    }
    phylabel     = "eth0"
    usage        = "ADAPTER_USAGE_MANAGEMENT"
    usage_policy = {}
    ztype        = "IO_TYPE_ETH"
  }
  io_member_list {
    assigngrp    = "eth1"
    cbattr       = {}
    cost         = 0
    logicallabel = "2nd_physical_intf"
    phyaddrs = {
      Ifname = "eth1"
    }
    phylabel     = "eth1"
    usage        = "ADAPTER_USAGE_UNSPECIFIED"
    usage_policy = {}
    ztype        = "IO_TYPE_ETH"
  }
  io_member_list {
    assigngrp    = "eth2"
    cbattr       = {}
    cost         = 0
    logicallabel = "3rd_physical_intf"
    phyaddrs = {
      Ifname = "eth2"
    }
    phylabel     = "eth2"
    usage        = "ADAPTER_USAGE_UNSPECIFIED"
    usage_policy = {}
    ztype        = "IO_TYPE_ETH"
  }
  io_member_list {
    assigngrp    = "eth3"
    cbattr       = {}
    cost         = 0
    logicallabel = "4th_physical_intf"
    phyaddrs = {
      Ifname = "eth3"
    }
    phylabel     = "eth3"
    usage        = "ADAPTER_USAGE_UNSPECIFIED"
    usage_policy = {}
    ztype        = "IO_TYPE_ETH"
  }
}
