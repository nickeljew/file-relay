# File-Relay

File-Relay is a caching service for accessing files in a short period by multiple services in inner system. It works like memcached for sharing data to multiple services.

- Max caching expiration: 10min
- Designed for small-size files, especially for image files, typically sizes from several KB to several MB
- Storage commands: set, add, replace
- Retrieval commands: get / gets; But retrieval of multiple values in a request is not supported


### Code Files Structure
```
---- main.go : main entry of file-relay server
 |-- filerelay/* : codes of ille-relay server
 |-- debug/* : simple debugging log library
 |-- client/* : test client for debugging
```

### TODO

- 