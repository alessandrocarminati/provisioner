# Sample RSA usage
 
Configuration may contain sensitive data that should not be left unattended
on the sidekick machine. In such cases, the provisioner offers a means to
utilize asymmetric RSA encryption to protect them. Here is an example usage
of the RSA encryption strategy.

This sample is updated at #783ea8efce04

```
$ ls
authorized_keys  config.json  id_rsa  id_rsa.pub  provisioner
$ cat config.json 
{
  "syslog_port": "514",
  "logfile": "provisioner.log",
  "tftp_directory": "./files/",
  "http_port": "8080",
  "ssh_serial_tunnel":{
    "port":"9090",
    "identity_fn": "id_rsa",
    "authorized_keys": "authorized_keys"
  },
  "ssh_monitor":{
    "port": "9091",
    "identity_fn": "id_rsa",
    "authorized_keys": "authorized_keys"
  },
  "serial_config": {
    "port": "/dev/ttyS0",
    "baud_rate": 115200
  },
  "monitor": {
  "pdu_type": "tasmota",
  "tasmota_host": "tasmota-8351ea-4586.hqhome163.com",
  "snmp_pdu_ctrl_oid": "1.2.3.4.5.6.7.8.9.10",
  "snmp_pdu_ctrl_host": "pdudevice.hqhome163.com",
  "snmp_pdu_ctrl_user": "someuser",
  "snmp_pdu_ctrl_on_val": "1",
  "snmp_pdu_ctrl_off_val": "0",
  "snmp_pdu_ctrl_on_type": "int",
  "snmp_pdu_ctrl_off_type": "int"
  }
}
$ ./provisioner -help
Usage of ./provisioner:
  -calfetch
    	fetch an element from calendar. Useful for 1st oauth authorization
  -config string
    	Config file name (default "config.json")
  -enc
    	interpret "config" as input file and "key" as private key file and outputs in config.rsa
  -genkeys
    	generates two new keypairs
  -help
    	Show help
  -key string
    	Key file name. if not given, the config isassumed plaintext

$ ./provisioner -genkeys
keys generated
$ ls
authorized_keys  config.json  id_rsa  id_rsa.pub  private  provisioner  public
$ ./provisioner -config config.json -enc -key public
config.rsa written
$ ls
authorized_keys  config.json  config.rsa  id_rsa  id_rsa.pub  private  provisioner  public
$ sudo ./provisioner -config config.rsa -key private
2024/03/20 09:20:31 Starting syslog service on port 514 -> provisioner.log 
2024/03/20 09:20:31 Starting TFTP service with rootdir:  ./files/
2024/03/20 09:20:31 Starting http service on port: 8080
2024/03/20 09:20:31 Starting tunnel SSH server on port 9090
2024/03/20 09:20:31 Starting monitor SSH server on port 9091
2024/03/20 09:20:31 Initialyzing monitor commands struct
```

