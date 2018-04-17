# hashtree
Introduction:
hashtree is a data dedeplication program that features client side encryption and S3 compatibility. I creates snapshots in time
of entire directory structures that can be duplicated or restored at a later date.

Files are never deleted on the server, they are just ommited from the file system structure in the snapshot - these files remain 
in the database and this means they will not be uploaded again at a later date.

This also means that if you backup files frequently the modified copies will be kept frozen in time and can be restored later. 

The program requires a ".htcfg" file with the following in the ~ (home) directory:
```
Url="127.0.0.1:9000" #any S3 endpoint
Port=9000 #deprecated use above version with the port in the url
Secure=false #boolean false or true (use HTTP or HTTPS)
Accesskey="S3 access key"
Secretkey="S3 secret key"
Enckey="your encryption key here, longer is better"
```
Be sure to print this file out. If you loose these details and you loose this config file, you will not be able to access your data ever again!

## To use the program:

**Initialise Repository:**
>		hashtree init <repository> <directory>
    
This will create the remote bucket and create an empty database. Please be aware that this can also overwrite an existing database, use with caution.

**List snapshots:**
>		hashtree list <repository>
    
This lists all available filesystem snapshots

**Deploy snapshot:**
>		hashtree pull <repository> <snapshot> <directory>

This will deploy a snapshot to a given directory. All paths will be created so the directory need not exist.

**Overwrite local files using remote repository:**
>       hashtree nuke <repository> <snapshot> <directroy>

This will overwrite any file with the same path as the remote repository, use with caution.

**Create snapshot:**
>		hashtree push <repository> <directory> 
    

This will create a new snapshot and upload any new files to the remote database. Each snapshot only takes up as much space as 
the new files and the size of the snapshot files (only a few 100kB).

Any problems please feel free to start a pull request. 

