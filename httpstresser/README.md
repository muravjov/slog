# httpstresser: general purpose HTTP stress tool

Usage:
```
$ ./httpstresser --help
Usage of httpstresser:
      --duration float       stress duration; < 0 means stress to be stopped with Ctrl+C (default -1)
      --keepalive            reuse TCP connections between different HTTP requests
      --request-number int   N >=0 means direct stress mode "for range N {StartJob()}" (default -1)
      --rps float            stress: request per second; < 0 means requesting without time throttling (default 5)
      --self                 start dummy http server at url
      --self-tls string      Custom tls config file, for --self server
      --timeout float        stress timeout; = 0 means no request timeout (default 5)
      --tls string           Custom tls config file: toml format, fields ca, cert, key, skip_verify, server_name
pflag: help requested
```

Stress 80 port at localhost, press Ctrl+C to stop stressing:
```
$ ./httpstresser http://localhost
2018/07/19 16:56:08 Starting stress:
^CStopped spawning jobs after 2.84 seconds
=======
Report

Counter Table:
%       Counter Name
100.00  14      "dial tcp [::1]:80: getsockopt: connection refused"

Totals:
Number of Requests:           14
Time Elapsed:                 2.84 seconds
Requests per Second:          4.93
Jobs Spawning per Second:     4.93
Success percent (HTTP 2xx):   0.00

Worst request duration, seconds:   0.002819
Mean  request duration, seconds:   0.001363
95ptl request duration, seconds:   0.002819
```

Stress during 2 seconds with 40 requests per seconds, using custom TLS config (for HTTPS):
```
$ ./httpstresser --timeout 10 --duration 2 --rps 40 --tls ./httpstresser-tls.toml https://api.my.service/path
2018/07/19 08:02:57 Starting stress:
Stopped spawning jobs after 2.00 seconds
=======
Report

Counter Table:
%       Counter Name
100.00  79      "2xx"

Totals:
Number of Requests:           79
Time Elapsed:                 9.12 seconds
Requests per Second:          8.66
Jobs Spawning per Second:     39.50
Success percent (HTTP 2xx):   100.00

Worst request duration, seconds:   8.423993
Mean  request duration, seconds:   0.960745
95ptl request duration, seconds:   0.968430
```
```
$ cat ./httpstresser-tls.toml
ca          = "/etc/puppetlabs/puppet/ssl/certs/ca.pem" # required, path to ca
cert        = "/etc/puppetlabs/puppet/ssl/certs/my.pem" # required, path to (cert, key) pair
key         = "/etc/puppetlabs/puppet/ssl/private_keys/my.pem" # required
skip_verify = true # optional, check server' certificate
server_name = "" # optional, what server name is required from server, defaults to host from URL
```

