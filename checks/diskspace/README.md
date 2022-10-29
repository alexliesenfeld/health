# Introduction 
The diskspace check uses the OS API to sample currently used filesystem storage and compare it to a threshold value. The threshold value is the *maximum* amount of *used* space permitted, expressed in bytes. For example, if you wanted to implement a check to determine if disk usage exceeded 2MB you would use a threshold value of 2097152. Currently only Unix is supported. 

# Multiple partitions/filesystems 
On Unix you can specify which filesystem to use by specifying a directory string. Whatever filesystem or partition that directory lives in will be checked. The default behavior (no directory provided to NewUnix()) uses whatever filesystem the current working directory is on. 
