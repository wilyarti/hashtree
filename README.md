# hashtree
# WARNING #
hashtree is currently being re-written in bad Go by the author. The previous version is still supported but no new features will be added.

# hashtree

S3 compatible data deduplication script written in Perl 

# Intro
Hashtree creates a data base containing a hash of each file in a directory and the files location. This way multiple copies of a file will not be uploaded saving remote storage. 

This also means when a file is updated the original file is still kept on the server but orphaned from the current database as the hash of the file will have changed. This will allow for (in the future - not implemeneted) a "snapshot" of a directory to be taken and reverted to or a snapshot of a single file with full history.

If there is a difference in the files then the user is prompted and a diff of the two files is created.

This program allows for multiple computers to have access to the same file system structure and allows for those files to be distrubuted over multiple computers regardless of local filesystem format.

# How to use
Download and configure s3cmd to use encryption and work with your chosen cloud provider.

Then either download the Linux binary "hashtree" or the Perl script "hashtree.pl" and run it.

If you use the Perl script you will need to install CPAN and all the missing dependancies.
$ pip3 install s3cmd
$ cpan File::Find Data::Dumper Digest::file Digest::SHA YAML YAML::Loader Array::Split File::Basename File::Path

You will also need:
rfcdiff
firefox

I will create Windows executables soon.

# Todo
~~I would like to rewrite this program in either Python or Golang and reduce all the dependancies into a single file. Either than or build a single installable Linux snap or Docker image to make installation simpler.~~

1.) Build a Windows and Mac OSX executable.

2.) A GUI is on the cards in the future.

3.) Eliminate the need for third party libraries. There is a bug in YAML::Loader with files that have multiple spaces. I will create a simple YAML formatter shortly.
4.) Write or implement a Perl base s3 storage backend. Currently using s3cmd.
