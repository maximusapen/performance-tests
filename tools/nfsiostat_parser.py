# *******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

import sys
import getopt
#
# This utility can be used to convert the rather unfriendly format that nfsiostat outputs
# into a CSV file for more easier viweing/plotting
#

def usage():
    """ Print information on how to use the module"""

    print('usage: python nfsiostat_parser.py -i <inputfile> -o <outputfile>')
    print('input: Raw nfsiostat output')
    print('output: CSV File with formatted output')

def parse_file(input_filename, output_filename):

    f = open(input_filename, 'r')
    out_f = open(output_filename, 'w')

    entryString = ''
    carryOn = True
    header = 'Mount point, Fileshare, Total op/sec, rpc bklog,, read_ops/s,read_kB/s, read_kB/op, read_retrans, read_avg_RTT_(ms), read_avg_exe_(ms),,write_ops/s,write_kB/s, write_kB/op, write_retrans, write_avg_RTT_(ms), write_avg_exe_(ms)\n'
    out_f.write(header)
    while (carryOn):
        # Find header line
        line = f.readline()
        if (line == ""):
            # End of File reached
            carryOn = False
            break;

        strIndex = line.find("mounted on")
        if strIndex != -1:
            split_line = line.split(" ")
            entryString = split_line[3].strip() + "," + split_line[0].strip() + ","
        else:
            continue
        # Blank line, then header
        f.readline(); f.readline()
        entryString = entryString + (f.readline()).strip() + ","
        entryString = entryString + (f.readline()).strip() + ",,"
        # read headers
        f.readline(); f.readline()
        for i in range(6):
             entryString = entryString + (f.readline()).strip() + ","
        # Write headers
        f.readline(); f.readline()
        entryString = entryString + ','
        for i in range(6):
             entryString = entryString + (f.readline()).strip() + ","

        entryString = entryString + "\n"
        out_f.write(entryString)
        entryString = ''

def main (argv):
    if (len(argv) < 1):
        usage()
        sys.exit(2)

    try:
        opts, args = getopt.getopt(argv,'hi:o:')
    except getopt.GetoptError:
        usage()
        sys.exit(2)

    inputfile = ''
    outputfile = ''
    for opt, arg in opts:
        if opt == '-h':
            usage()
            sys.exit()
        elif opt  == '-i':
            inputfile = arg
        elif opt == '-o':
            outputfile = arg

    print ('Parsing input file "' + inputfile + '"')

    parse_file(inputfile, outputfile)
    print ('Parsing complete, output written to "' + outputfile + '"')

if __name__ == "__main__":

    main(sys.argv[1:])
