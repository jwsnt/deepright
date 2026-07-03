#!/bin/sh
exec java -server -Xms4g -Xmx4g -XX:MaxGCPauseMillis=200 -XX:InitiatingHeapOccupancyPercent=40 -XX:+UseStringDeduplication -XX:+UnlockExperimentalVMOptions -XX:+UseZGC -XX:+TieredCompilation -XX:CompileThreshold=1000 -jar /deepright-1.0.jar
