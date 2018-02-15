# WARNING #
This script is completely unsupported and comes with no warranty!

I am currently building the program to learn programming. It was written in Perl but now I am completely re-writing it in Go.

# Hashtree
File based data-deduplication with S3 compatibility.

The Perl script hashtree.pl has the most features.

1.) File based deduplication
2.) Client side encryption
3.) Duplication across multiple environments
4.) File diffing 

Hashtree will only upload one copy of a file, but will use it's database to recreate the entire filesystem structure.

This means you can point it at a folder and it will only upload files that are missing on the cloud. It is handy if you are messy like me but want to ensure you have everything backed up.

It can scale and I have used it to process 1TB of data.

The new Go implementation will reduce dependancies and increase speed.
