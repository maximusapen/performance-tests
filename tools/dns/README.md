# Kubernetes DNS Perf-test

## Test Quick Start

DNS test based on https://github.com/kubernetes/perf-tests/tree/master/dns.

Clone the dns GIT.  Overlay files in tools/dns on top to override/add to the
Kubernetes GIT for our DNS test using python3.

Besides installing python3 and all the python dependencies, you also need to
install numpy.

    pip3 install numpy

The DNS perf-test has option to create DNS test server pod (kubedns or coredns).
In this case, test will create a client pod and a server pod.

We test our cluster DNS with option --use-cluster-dns and only client pod is created.

To do a quick test, following command(s) will run test for 60s with result in out/out-<numbers>.
A symbolic link `latest` always link to the most recent test result.

    mkdir out
    python3 py/run_perf.py --params params/coredns/test.yaml --out-dir out --use-cluster-dns

Test will create a `dns-perf-client` pod and 20 test services named `test-svc<number>`.

## Test further details

### YAML warning

Your version of yaml dependency may have these warnings:

    YAMLLoadWarning: calling yaml.load() without Loader=... is deprecated, as the default Loader is unsafe. Please read https://msg.pyyaml.org/load for full details.

To remove these warnings, add `Loader=yaml.FullLoader` to all the `yaml.load`.  Example,

    yaml.load(out)

becomes

     yaml.load(out, Loader=yaml.FullLoader)

The perf clients currently do not have these warnings.  It failed to run if Loader is specified.  The yaml change may be required in future with client update.

### Parameter files

Theese coredns parameters in perf-test-dns/params/coredns/*.yaml are for test dns server which we don't use in our testing:

    # cpu limit for coredns, null means unlimited.
    coredns_cpu: [null]
    # size of coredns cache. Note: 10000 is the maximum. 0 to disable caching.
    coredns_cache: [0]

In theory we can just use the parameter files in perf-test-dns/params/nodelocaldns/*.yaml.

## NodeLocal DNS Cache testing

The enable_nodelocal_dns_cache.sh and disable_nodelocal_dns_cache.sh enables/disables
NodeLocal DNS Cache for testing.

Once cluster is configured with/without NodeLocal DNS Cache using `*_nodelocal_dns_cache.sh`.
Scripts are provided here to run the test and organize the test results for different
test config.

Test coredns:

    ./run3.sh coredns

Test NodeLocal DNS cache:

    ./run3.sh nodelocaldns

To repeat test with client on and off node with coredns.  Cordon nodes to achieve this.
You need to leave 2 uncordoned nodes for test to run even though we only run with --use-cluster-dns
options with one client pod and no dns server pod.

With client on coredns node, add oncoredns or any reportDir name you chose.  One example below.

Test coredns:

    ./run3.sh coredns oncoredns

Test NodeLocal DNS cache:

    ./run3.sh nodelocaldns oncoredns

With client off coredns node, add offcoredns or any directory name you chose

Test coredns:

    ./run3.sh coredns offcoredns

Test NodeLocal DNS cache:

    ./run3.sh nodelocaldns offcoredns

You can also set reportDir as test1, test2, test3....etc.
