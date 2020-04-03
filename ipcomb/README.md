# IPCombine
Combine domain, ip, port

## Usage
```
ipconb file1.txt file2.txt
```

#### file1.txt
##### Format: domain,ip
```
google.com,127.0.0.1
```

#### file2.txt
##### Format: ip:port
```
127.0.0.1:8080
```

#### Output
```
google.com:8080
```