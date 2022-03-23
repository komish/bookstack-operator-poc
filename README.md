# BookStack Operator

This is an incomplete proof of concept Kubernetes operator managing
[BookStack](https://www.bookstackapp.com/).

This is not ready for any use other than hacking in its current state, for
reasons such as hard-coded testing credentials, and focusing on reaching
a working state in Docker Desktop's included Kubernetes implementation.

This leverages [linuxserver.io's BookStack
images](https://docs.linuxserver.io/images/docker-bookstack) which aren't
necessarily designed to work in Kubernetes contexts.

Regardless of the current state of the project, the operator implementation
features individual controllers per resource grouping,
[OperatorSDK](https://sdk.operatorframework.io/) along with the user of the
[subreconcilers](https://github.com/opdev/subreconciler) library for easy-to-read
reconciliation results.

All controllers create the resource stack necessary for BookStack to function.

The latest unresolved issue revolves around the bookstack-db container which is
unable to initialize the database due to some issue writing to the volume mount.
