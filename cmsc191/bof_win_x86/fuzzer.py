#fuzzer.py
#n must be adjusted so that the return 
#address, as shown in gdb, is not overwritten by As (0x41). 

n = 100

payload = 'A' * n

print payload

try:
    f = open("C:\\simple_exploit_payload.bin", "wb") # Exploit output will be written to C directory
    f.write(payload)                        # Write entirety of buffer out to file
    f.close()                           # Close file
    print "\nExploit written successfully!"
    print "Buffer size: " + str(len(payload)) + "\n" # Buffer size sanity check to ensure there's nothing funny going on
except Exception, e:
    print "\nError! Exploit could not be generated, error details follow:\n"
    print str(e) + "\n"
