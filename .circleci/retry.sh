#/bin/bash
attempts=$1
shift
for (( c=1; c<=$attempts; c++ ))
do
  echo attempt $c at \"$@\":
  $@
  if [ $? -eq 0 ]
  then
    exit 0
  fi
done

echo failed $attempts times attempting to run \"$@\"
exit 1