# Carrier Utils
A location for various tools to be used on a carrier

The tools generally require access to `kubectl` for the carrier. Some also require acces to a `/etc/hosts` file that contains entries for the worker nodes on the carrier (so they can perform IP to hostname mapping). Some may need to be run on the carrier master. Each script should declare in the comments which dependencies they have.
