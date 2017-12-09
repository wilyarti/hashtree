# hashtree
S3 compatible data deduplication script written in Perl

# Intro
Hashtree creates a data base containing a hash of each file in a directory and the files location. This way multiple copies of a file will not be uploaded saving remote storage. 

This also means when a file is updated the original file is still kept on the server but orphaned from the current database as the hash of the file will have changed. This will allow for (in the future - not implemeneted) a "snapshot" of a directory to be taken and reverted to or a snapshot of a single file with full history.

If there is a difference in the files then the user is prompted and a diff of the two files is created.

This program allows for multiple computers to have access to the same file system structure and allows for those files to be distrubuted over multiple computers regardless of local filesystem format.

# How to use
Enter your S3 compatible storage object store keys and host URL in the hashtrees3upload and hashtrees3download files. Make these files executable and place them in your path. 

$ chmod +x hashtrees3upload hashtrees3download
$ chmod 700 hashtrees3upload hashtrees3download

Then enter the name of your "bucket" and point the the working directory:

$ hashtree mybucket ~/files

If you get errors about missing dependancies install them using cpan:

$ cpan File::Find Data::Dumper Digest::file YAML Array::Split File::Basename File::Path 

If you get errors with Python:

$ pip3 install boto3

Follow the prompts of the script.

# Todo
I would like to rewrite this program in either Python or Golang and reduce all the dependancies into a single file. Either than or build a single installable Linux snap or Docker image to make installation simpler.

A GUI is on the cards in the future.
