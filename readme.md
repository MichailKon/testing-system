# Testing system

This is a fast testing system for programming tasks.

## Installation

The testing system can be used at any linux repository. 

You will need go 1.24 to build the testing system. 
You will need postgres for database. 
You will also need [isolate](https://github.com/ioi/isolate) for code isolation.

To install testing system, download the repository. Then run command:

```shell
go build ./
```

To run the testing system, use command:

```shell
./testing_system <config.yaml file path>
```

There are two ways to configure testing system:

- Single process run. For this configuration, use the file `configs/config-single-example.yaml` as a base for config. You can read the comments in config to configure testing system correctly.
- Multiple server run. 
For this configuration, there should be master server where the master component of testing system is set up, 
and there can be multiple invoker servers for invokers.
For config files, use `config/config-master-example.yaml` for master server config 
and `config/config-invoker-example.yaml` for invoker server config.

### Client configuration

For client build, use command:

```shell
go build ./clients
```

To run the client, run compiled binary with config as a first argument. 
The config can be produced from `configs/client-config-example.yaml`.

### Polygon import

To import problems from polygon, you can use `tools/polygon_importer`.
It should be run with the following command:
```shell
polygon_importer <testing system config path> <polygon ID of problem> <polygon API key> <polygon API secret>
```

