#!/usr/bin/env perl

=for comment
Copyright <2017> <Wilyarti Howard>
 
Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
 
1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
 
2. Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentatio
n and/or other materials provided with the distribution.
 
3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products derived from this software w
ithout specific prior written permission.
 
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE 
GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRIC
T LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SU
CH DAMAGE.
                
=cut

use warnings;
use strict;
use File::Find;
use Data::Dumper;
use Digest::file qw(digest_file_hex);
use Digest::SHA;
use YAML qw(Dump);
use YAML::Loader;    # qw(Loader);
use Array::Split qw( split_into );

#for creating filepaths
use File::Basename;
use File::Path qw/make_path/;
use threads;
use threads::shared;
use constant NUM_THREADS      => 12;
use constant NUM_DOWN_THREADS => 4;

my $Yflag = 0;
my $Nflag = 0;
our $DLflag = 0;
our $ULflag = 0;
our $Sflag  = 0;

binmode( STDOUT, ":utf8" );

our $repo = "";
our $path;
our @filelist;
our %remotetable;
our %localtable;
our %dlerrortable;
our %ulerrortable;
our @threads;
our $count = 0;
share(%localtable);
share(%ulerrortable);
share(%dlerrortable);
share($count);

sub main {

    #open local database file
    open( FH, ">$path/.$repo.hsh" )
      or die "Can't open hash file. $path/.$repo.hsh";

    print "\nDownloading file index: ";

    my $code = dwld( "$repo.hsh", "$path/.$repo.master.hsh", $repo );
    if ( $code != 0 ) {
        print "Error retrieving remote database.\n";
        print "Choose (c) create new (a) abort: ";

        my $ans = <STDIN>;
        chomp($ans);
        if ( $ans eq "c" ) {
            print "Creating new repo.\n";
        }
        else { die "Aborted.\n"; }
    }
    my $indexfile;
    my $localindexfile;
    {
        open( my $fh, "<$path/.$repo.master.hsh" )
          or warn "Can't open hash file. $path/.$repo.master.hsh";
        local $/ = undef;
        $indexfile = <$fh>;
        close $fh;
    }
    {
        my $scandir = 1;
        if ( -f "$path/.$repo.hsh" ) {
            print
"Hash database already exist. Load existing database or rescan directory?\n";
            print "Load (l) Rescan (r): ";
            my $ans = <STDIN>;
            chomp($ans);
            if ( $ans eq "l" ) {
                $scandir = 0;
            }
        }
        if ( $scandir == 1 ) {

            #create local file list
            find( \&wanted, $path, );
            #### Find all files in directory provided
            #backup original hash as split_hash() deletes it
            my $localcount = @filelist;
            print "Processing $localcount local files.\n";

            #Split array into chunks for threads
            my @subfilelist = split_into( NUM_THREADS, @filelist );

            for my $arrayref (@subfilelist) {
                push @threads,
                  threads->create( \&hashfiles, $arrayref, \%localtable );
            }

            #Join threads to share their data
            for (@threads) {
                $_->join();
            }
        }
        else {

            {
                open( my $fh, "<$path/.$repo.hsh" )
                  or warn "Can't open hash file. $path/.$repo.hsh";
                local $/ = undef;
                $localindexfile = <$fh>;
                close $fh;
            }

            #load YAML into hash
            my $loader = YAML::Loader->new;
            my %temptable;
            eval { %temptable = $loader->load($localindexfile); };
            if (@_) { warn "FIX ME @_"; }
            while ( ( my $hash, my $filearray ) = each %remotetable ) {
                for ( @{$filearray} ) {
                    no warnings 'experimental';
                    push @filelist, $_;
                    $localtable{$_} = $hash;

                }
            }
        }
    }

    #load YAML into hash
    my $loader = YAML::Loader->new;

    #add error handling here
    eval { %remotetable = $loader->load($indexfile); };
    if (@_) { warn "FIX ME @_"; }

    #Setup and process databases
    my %localdb;
    my %remotedb;
    my %uploadlist;
    my %downloadlist;
    my %conflictlist;

    #add path
    while ( ( my $hash, my $filearray ) = each %remotetable ) {
        for ( @{$filearray} ) {
            $_ = $path . $_;
        }
    }

    #localtable is a flat hash
    #compare local file list with remote file list
    #check for differences and create conflicts
    #other wise upload new files
    while ( ( my $path, my $checksum ) = each %localtable ) {

        #create hash of arrays and push each duplicate file to array
        my $match = 0;
        for my $files ( @{ $remotetable{$checksum} } ) {
            if ( $files eq $path ) {
                $match = 1;
            }
        }

        #unless file path and checksum match add to upload list
        #any conflicts will be checked for before upload
        if ( !$match ) {

            #print "[A] $path\n";
            $uploadlist{$path} = $checksum;
        }
    }

    #remote table is an hash of arrays
    while ( ( my $hash, my $filearray ) = each %remotetable ) {
        for ( @{$filearray} ) {

            #download unless file path exists (regardless of file checksum)dd
            if ( !defined $localtable{$_} ) {
                print "[D] $_\n";
                $downloadlist{$_} = $hash;

            }
            else {
                if ( $localtable{$_} ne $hash ) {

                  #add two different hashes to list refered to by file path
                  #as the key. then later the array can be index to download the
                  #file and produce diffs of conflicting files.
                    $conflictlist{$_} = [ $localtable{$_}, $hash ];

                    #delete from upload list
                    #may be added later depending on users examination of diff
                    delete $uploadlist{$_};
                }
                else {
                    print "[V] $_\n";
                    push( @{ $localdb{$hash} }, $_ );
                }
            }

        }
    }

    #prompt for download
    #store file user choses in the userdllist
    #discard old downloadlist
    my $userdllist = prompt( "dl", \%downloadlist );

    #resolve conflict between local and remote files
    my $userullist = prompt( "ul", \%conflictlist );
    while ( ( my $filepath, my $hash ) = each %$userullist ) {

    #check if file is new (over written by downloaded file in promt step above),
    #if so add new downloaded file to and skip upload step
        if ( $localtable{$filepath} ne $hash && $hash ) {
            push( @{ $localdb{$hash} }, $filepath );
        }
        else {
            $uploadlist{$filepath} = $hash;
        }
    }

    #split hashes into an array of hashes to be threaded
    my $dlsize = ( ( keys %$userdllist ) / NUM_DOWN_THREADS ) + 1;
    my $ulsize = ( ( keys %uploadlist ) / NUM_DOWN_THREADS ) + 1;
    my ( %dllist, %ullist ) = ( %$userdllist, %uploadlist );
    my @dlhash = split_hash( $dlsize, $userdllist );
    my @ulhash = split_hash( $ulsize, \%uploadlist );

    #start uploader
    my @ulthreads;
    for my $hashref (@ulhash) {
        push @ulthreads,
          threads->create( \&uploader, $hashref, \%ulerrortable );
    }

    #Join threads to share their data
    for (@ulthreads) {
        $_->join();
    }

    #start downloader
    my @dlthreads;
    for my $hashref (@dlhash) {
        push @dlthreads,
          threads->create( \&downloader, $hashref, \%dlerrortable, $Sflag );
    }

    #Join threads to share their data
    for (@dlthreads) {
        $_->join();
    }

    # warn of errors - not in scope of script to fix.
    my $dlerrorcount = keys %dlerrortable;
    my $ulerrorcount = keys %ulerrortable;

    # abort or cleanup not uploaded files
    if ($ulerrorcount) {
        print "Warning $ulerrorcount files did not upload successfully.\n";
        print "Please choose (a) abort (o) orphan: ";
        my $ans = <STDIN>;
        chomp($ans);
        if ( $ans eq "o" ) {
            while ( ( my $path, my $checksum ) = each %ulerrortable ) {
                delete $ullist{$path};
                delete $ulerrortable{$path};
            }
        }
        else { die "Aborted.\n"; }
    }

    # abort or cleanup no downloaded files
    if ($dlerrorcount) {
        print "Warning $dlerrorcount files did not download successfully.\n";
        print "Please choose (a) abort (o) orphan: ";
        my $ans = <STDIN>;
        chomp($ans);
        if ( $ans eq "o" ) {
            while ( ( my $path, my $checksum ) = each %dlerrortable ) {
                delete $dllist{$path};
                delete $dlerrortable{$path};
            }
        }
        else { die "Aborted.\n"; }
    }

    # add downloaded files to localdb
    while ( ( my $path, my $checksum ) = each %dllist ) {
        if ( !$path ) { warn "no path step 1\n"; }
        unless ( !$path ) { push( @{ $localdb{$checksum} }, $path ); }

    }

    # add uploaded file to localdb
    while ( ( my $path, my $checksum ) = each %ullist ) {
        if ( !$path ) { warn "no path step 2\n"; }
        unless ( !$path ) { push( @{ $localdb{$checksum} }, $path ); }
    }

    #remove path from database
    while ( ( my $hash, my $filearray ) = each %localdb ) {
        for ( @{$filearray} ) {
            $_ =~ s/$path//;
        }
        @{$filearray} = uniq( @{$filearray} );

    }

    my $date = `date +%d-%m-%Y-%H:%M:%S`;
    chomp($date);

    my $indexcount = keys %localdb;
    print "Uploading new index containing $indexcount unique files? ";
    my $ans = <STDIN>;
    chomp($ans);
    if ( $ans eq "n" ) {
        die "Aborting\n";

    }

    print FH Dump(%localdb);
    close FH;
    upld( "$path/.$repo.hsh", "$repo.$date.hsh", $repo );
    upld( "$path/.$repo.hsh", "$repo.hsh",       $repo );

}

sub wanted {
    if ( -f $_ ) {
        unless ( $_ eq ".$repo.hsh" || $_ =~ ".$repo.master.hsh" ) {
            push @filelist, $File::Find::name;
        }
    }
}

sub hashfiles {
    my ( $filelist, $localhash ) = @_;
    my $id = threads->tid;
    for my $path (@$filelist) {
        my $sha256sum;
        eval { $sha256sum = digest_file_hex( $path, "SHA-256" ); };
        $|++;

        $count++;
        print "\rHashing file $count";
        warn $@ if $@;
        unless ( $@
            && $sha256sum ne
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" )
        {
            $$localhash{$path} = $sha256sum;
        }
    }
    return $localhash;
}

sub split_hash {
    my ( $x, $hash ) = @_;
    my @hashes;
    while (%$hash) {
        my @k = keys %$hash;
        push @hashes, { map each %$hash, 1 .. $x };
        delete @{$hash}{ keys %{ $hashes[-1] } };
    }
    return @hashes;
}

sub uploader {
    my ( $hash, $errorhash ) = @_;
    my $id = threads->tid;
    my %unique;
    my $code;

    while ( ( my $filepath, my $filehash ) = each %$hash ) {
        $unique{$filehash} = $filepath;
    }
    while ( ( my $filehash, my $filepath ) = each %unique ) {
        if ( $filepath eq $path ) {
            warn "Empty value! $path \n";
        }
        else {
            $code = upld( $filepath, $filehash, $repo );
            if ( $code ne 0 ) {
                print "(E!) $filepath\n";
                $$errorhash{$filepath} = $filehash;
            }
        }
    }
    return $errorhash;

}

sub prompt {
    my ( $opt, $filehash ) = @_;
    my $filecount = keys %$filehash;
    if ( $filecount == 0 ) { return; }
    if ( $opt eq "dl" ) {
        print "There are $filecount remote files missing locally.\n";
        print
"Choose (d) download all, (c) choose for each, (s) skip download but keep in database, (r) remove remote files from database: ";
        my $ans = <STDIN>;
        chomp($ans);
        my $Dflag = 0;
        if ( $ans eq "s" ) {
            print "Skipping download. Keeping remote files in database.\n";
            $Sflag = 1;
        }
        elsif ( $ans eq 'd' ) {
            $Dflag = 1;
        }
        elsif ( $ans eq 'r' ) {
            print "Type YES to confirm:  ";
            my $ans = <STDIN>;
            chomp($ans);
            if ( $ans ne "YES" ) {
                die "Aborting...\n";
            }
            while ( ( my $path, my $hash ) = each %$filehash ) {
                print "Deleting $$filehash{$path}\n";
                delete $$filehash{$path};

            }

        }
        else {
            while ( ( my $path, my $hash ) = each %$filehash ) {
                if ($Dflag) {

                    print "(D) $path\n";
                }
                else {
                    print "Download $path with $hash (y/n): ";
                    my $ans = <STDIN>;
                    chomp($ans);

                    if ( $ans eq "n" ) {
                        print
"If you answer yes it will delete the file from the remote database permentantly.\n";
                        print "Continue with delete? (y/n): ";
                        my $ans = <STDIN>;
                        chomp($ans);
                        if ( $ans eq "y" ) {
                            print "Deleting $$filehash{$path}\n";
                            delete $$filehash{$path};
                        }
                    }
                }
            }
        }
        return $filehash;
    }
    elsif ( $opt eq "ul" ) {
        my %returnhash;
        print
          "There are $filecount conflicts between local and remote files.\n";
        print
"(d) replace all local files (u) replace all remote file (c) choose for each by diff: ";
        my $ans = <STDIN>;
        chomp($ans);
        my $Dflag = 0;
        my $Uflag = 0;
        if    ( $ans eq "d" ) { $Dflag = 1; }
        elsif ( $ans eq "u" ) { $Uflag = 1; }

        while ( ( my $filepath, my $arrayref ) = each %$filehash ) {
            if ($Dflag) {
                dwld( $$arrayref[1], $filepath, $repo );
                $returnhash{$filepath} = $$arrayref[1];

            }
            elsif ($Uflag) {
                $returnhash{$filepath} = $$arrayref[0];
                print "[U] $filepath\n";
            }
            else {
                print "Remote hash: $$arrayref[1] Local hash: $$arrayref[0]\n";
                print "Making diff: \n";
                my $tmpfile = "/tmp/" . $$arrayref[1];
                dwld( $$arrayref[1], $tmpfile, $repo );
                system
qq [ rfcdiff --stdout '$filepath' '$tmpfile' > '$tmpfile.diff.html' ];
                system( "firefox", "$tmpfile.diff.html" );
                print
                  "Which version would you like to keep: (l)ocal or (r)emote: ";
                my $ans = <STDIN>;
                chomp($ans);

                if ( $ans eq "r" ) {
                    dwld( $$arrayref[1], $filepath, $repo );
                    $returnhash{$filepath} = $$arrayref[1];

                }
                else {
                    $returnhash{$filepath} = $$arrayref[0];
                }
            }

        }
        return \%returnhash;
    }

}

sub downloader {
    my ( $hash, $errorhash, $Sflag ) = @_;
    my $id = threads->tid;
    my $code;
    while ( ( my $filepath, my $filehash ) = each %$hash ) {
        if ( $filepath eq $path ) {
            warn "Empty value! $filepath\n";
        }
        elsif ( $Sflag == 1 ) {
            print "(S) $filepath\n";
            $code = 0;
        }
        else {
            $code = dwld( $filehash, $filepath, $repo );
            if ( $code ne 0 ) {
                print "(E!) $filepath\n";
                $$errorhash{$filepath} = $filehash;
            }
        }

    }
    return $errorhash;

}

sub upld {
    my $source      = $_[0];
    my $dest        = $_[1];
    my $bucket_name = $_[2];
    my $code        = 0;
    print "(U) $source => $dest\n";
    system qq [ s3cmd -e -q put '$source' s3://'$bucket_name'/'$dest' ];
    $code = $?;
    return $code;

}

sub dwld {

    my $source      = $_[0];
    my $dest        = $_[1];
    my $bucket_name = $_[2];
    my $dir         = dirname($dest);
    my $code        = 0;
    eval { make_path($dir); };

    if ($@) {
        print "Can't make directory path for file $dest!\n";
        $code = -1;
    }
    else {
        print "(D) $source => $dest\n";
        system
          qq [s3cmd -e -q --force get s3://'$bucket_name'/'$source' '$dest'];
        $code = $?;
        my $shasum;
        eval { $shasum = digest_file_hex( $dest, "SHA-256" ); };
        warn $@ if $@;

        if ($@) {
            $code = -1;
        }
        elsif ( $shasum ne $source && $source ne "$repo.hsh" ) {
            warn
"Checksum mismatch on file $dest with hash of\n\t$shasum vs\n\t$source\n";
            print "Omitting $dest from database - chechsum mismatch!";
            $code = -1;
        }
    }
    return $code;

}

sub uniq {
    my %seen;
    grep !$seen{$_}++, @_;
}

my $errormsg = "Missing DIRECTORY!\nUsage: hashtree REPOSITORY DIRECTORY\n";
if ( !$ARGV[1] ) {
    die $errormsg;
}
elsif ( -d $ARGV[1] ) {
    $repo = $ARGV[0];
    $path = $ARGV[1];
    $path = $1 if ( $path =~ /(.*)\/$/ );
    my $output = `s3cmd --version`;
    my $code   = $?;
    if ( $code != 0 ) { die "Error: Please install s3cmd atleast version 2+"; }
    elsif ( $output !~ "2." ) {
        die "Error this script requires verson 2.0.1 or greater!";
    }
    main();
}
else { die $errormsg; }
