# Prometheus deleted mapping exporter

This repository provides code for a Prometheus metrics exporter that reports executable memory mappings 
made by processes to files that have been deleted in the meantime.
This is useful to detect processes that still make use of outdated executables and libraries.
This exporter provides a separate metric for every file that has been deleted and still has mappings, 
where the value of the metric corresponds to number of mappings present.
It extracts these metrics from `/proc/<nr>/maps`.
The path to the mount directory of `proc` can be specified by setting the `-deletedmapping.proc-path` flag.