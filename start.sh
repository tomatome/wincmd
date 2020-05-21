#!/bin/sh
for j in {1..1000000};
do
for i in {1..100};
do
queue="normal"
n=5
a=$(( $i % 4 ))
b=$(( $i % 7 ))
if [ $a == 0 ];then
queue="low"
n=3
elif [ $b == 0 ];then
queue="high"
n=1
fi
su jhadmin -c "jsub -q ${queue} -E hostname -Ep whoami sleep $n";
sleep 1
done
done
