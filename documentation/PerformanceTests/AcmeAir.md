# AcmeAir Tests

## Objective

* This benchmark emulates a fictional airline called 'Acme Air'. The benchmark represents multiple users concurrently interacting with the airline. The benchmark code is here: <https://github.com/blueperf>.

## Description

* The customer requests are generated using a jmeter http client. The requests are used to to book, cancel or review flights. Each request interacts with one or more of the benchmark's microservices:
  * authorization
  * customer service
  * booking
  * flight
  * main service

* The microservices run in pods in the Cruiser and interact with associated databases.

## Charts

Grafana chart:  `_Acmeair` - metrics `acmeair`

* Higher numbers show better performance for the throughput charts
* Lower numbers are better for the latency charts.

## Details
