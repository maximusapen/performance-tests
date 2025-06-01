# Generate Results Alerts

This tool is used to alert the performance squad when results from the automated performance test runs are poor. The tool reads the test results from the Influx DB and compares them against configured alert values. Any alerts generated are displayed via the standard output and squad members can be automatically alerted via Slack notifications.

## Running

./alerting [-verbose] [-debug]  

Usage of ./alerting:  
  - -debug  
      Debug output  
  - -verbose  
      Detailed logging output  

## Configuration

Configuration Data is stored in yaml format. Example can be found [here](https://github.ibm.com/armada-performance/armada-performance/blob/alerting/metrics/alerting/config/perf-alerts.yaml)  

### influxdb
- host  
Hostname or IP Address where the Influx database is running (e.g. `127.0.0.1`)
- port  
Port on which the Influx database is listening (e.g. `8086`)
- database  
Name of influxdb database (e.g. `ArmadaPerf`)
- username  
InfluxDB username (e.g. `admin`)
- timeout  
Connection timeout in seconds (e.g. `30`)

### slack  
- enabled  
Boolean indicating whether alert summary data should be sent to Slack
- channel  
Slack channel name to which alert summary data should be sent (typically `armada-perf-alerts`)  

### options  
- history
  * count  
  Maximum number of results to be processed for an individual test  
  * days  
  Maximum age of results to be processed for an individual test
  * current  
  Nunber of results to be considered the latest test data. If greater than 1, the mean of the results is used. This can be used to require that any results that may trigger an alert are repeatable.  
  * minimum  
  Minimum number of results required for generating informational or z-score alerts. This can be used to prevent historical based alerts from being generated when there are a low number of previous results for a particular test.

- leniency  
Used to identify alert thresholds that are unduly lenient. Specify a [Z-Score](https://www.statisticshowto.com/probability-and-statistics/z-score/#Whatisazscore). Any threshold that is more than this number of standard deviations away from the mean will be highlighted and should be considered as a candiate for manual adjustment.  

- verbose  
Boolean indicating whether additional output should be generated. If set to false, only the alerts will be displayed. Can be overidden on the command line.  

### environments  
A map containing test environment information. The map key should be a description of the environemnt (e.g. `"IKS on Classic"`)  
- carrier  
The name of the carrier used for this test environment
- machineType  
An array of machine types associated with this test environment  (e.g. `[b3c.4x16, bx2.4x16]`)
- kubeVersion  
An array of Kubernetes versions associated with this test environment (e.g. `[1_19, 1_20]`)
- owner  
  * name  
  Name of the person resposible for the test environemnt
  * slack  
  Member ID of environment owner
  * notify  
  Boolean indicating whether a DM should be sent to the environment owner

### tests  
An array of tests to be processed, containing alert configuration data
- name  
Test name (e.g. `httpnodeport`)
- environment  
  * kubeVersion  
  An array of Kubernetes versions to which the alerts are applicable
  * operatingSystem
  An array of operating systems to which the alerts are applicable
  * alerts  
    * name  
    Test metric name (e.g. `nodes5_replicas3_threads_20_singlezone_throughput`)
    * limitType  
    `floor` or `ceiling`
    * thresholds  
      One set of theshold values for each KubeVersion specified
      * warn
      Numerical value above or below which a warning alert will be generated
      * error  
      Numerical value above or below which an error alert will be generated
      * zscore  
      Number of standard deviations away from the historical mean above or below which an alert will be generated.

## Output  

Four types of alerts may be generated. 
- **Warning**   
Indicates that a test result has triggered the simple threshold for a Warning alert  
E.g.   
<span style="color:yellow">	
ALERT - WARNING  
	Owner: Richard  
	Environment: carrier3_stage, MachineType: bx2.4x16, Version: 1.19  
	Test: acmeair_istio - nodes5_threads_60_singlezone_throughput  
	Timestamp: 2021-04-21 00:26:45 +0000 UTC, Threshold: 2000.0, Result: 1757.4</span>  

- **Error**  
Indicates that a test result has triggered the simple threshold for a Warning alert  
E.g.   
<span style="color:red">	
ALERT - ERROR  
	Owner: Janet  
	Environment: carrier4_stage, MachineType: b3c.4x16, Version: 1.19  
	Test: k8s_netperf - iperf_TCP_Remote_VM_using_Virtual_IP_calico_singlezone_max  
	Timestamp: 2021-05-05 20:53:58 +0000 UTC, Threshold: 3000.0, Result: 2547.0</span>  

- **ZScore**  
Indicates that a test result is more than the specified number of standard deviations away from the historical mean  
<span style="color:magenta">
ALERT - Z-SCORE
	carrier3_stage: MachineType: bx2.4x16, Version: 1.19
	httpvpcapplicationloadbalancer - nodes5_replicas3_threads_20_singlezone_average
	2021-04-21 03:30:57 +0000 UTC : Threshold: 2.0, Result: 2.6</span>  

- **Information**  
Indicates that the specified alert thresholds are unlikely to be triggered based on historical results and should be considered for adjustment. See leniency above.  
E.g.  
<span style="color:green">	
ALERT - INFORMATION  
	Owner: Richard  
	Environment: carrier3_stage, MachineType: bx2.4x16, Version: 1.19  
	Test: acmeair_istio - nodes5_threads_60_singlezone_percentile99  
	Timestamp: 1970-01-01 00:00:00 +0000 UTC, Threshold: 3.0, Result: 9.7</span>
