# Registry Tests

## Objective

* These tests measure the time taken to upload and download images from the Regional and Global registries.

## Description

* The tests run in containers on production Cruisers in a number of regions around the world e.g. US-South, UK, Germany, Sydney, Tokyo. In each case the test pushes and pulls a 50MB image (single layer) to the local regional registry and the global registry. It then pulls the Hyperkube 1.8.6 image (512MB but with multiple image layers) from the regional registry.  Tests are run twice a day.

## Charts

Grafana chart:  `_Registry`

* The charts show the time taken to push and pull images plotted against the date of the test.
  * Lower numbers show better performance.

## Details
