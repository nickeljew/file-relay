# File-Relay

File-Relay is a caching service for accessing temporary files in a short period by multiple services in internal system.
It works like memcached for sharing data to multiple services.

- Max caching expiration: 10min
- Designed for small-size files, especially images, typically those sizes from 1KB to 10MB
- Commands:
  - Storage commands: set, add, replace
  - Retrieval commands: get / gets 
  (But only support one value in a retrieval request)


## Code Files Structure
```
---- main.go : main entry of file-relay server
 |-- filerelay/* : codes of ille-relay server
 |-- debug/* : simple debugging log library
 |-- client/* : test client for debugging
```

## TODO

- Cluster mode
- Persistence-on-demand: only on demand by client
  - Using plugins to support external persistence, such as Ceph, S3, etc.
  - Support persistence command: persist


## License

[MIT](http://www.opensource.org/licenses/mit-license.php)