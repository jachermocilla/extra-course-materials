// Monitor class for reader-writer synchronization
class ReaderWriterMonitor {
    private int readersActive = 0;
    private int writersActive = 0;
    private int writersWaiting = 0;
    
    // Reader requests access
    public synchronized void startRead() throws InterruptedException {
        // Wait while a writer is active or writers are waiting
        while (writersActive > 0 || writersWaiting > 0) {
            wait();
        }
        readersActive++;
    }
    
    // Reader releases access
    public synchronized void endRead() {
        readersActive--;
        // If no more readers, notify waiting writers
        if (readersActive == 0) {
            notifyAll();
        }
    }
    
    // Writer requests access
    public synchronized void startWrite() throws InterruptedException {
        writersWaiting++;
        // Wait while any readers or writers are active
        while (readersActive > 0 || writersActive > 0) {
            wait();
        }
        writersWaiting--;
        writersActive = 1;
    }
    
    // Writer releases access
    public synchronized void endWrite() {
        writersActive = 0;
        // Notify all waiting threads (readers and writers)
        notifyAll();
    }
}

// Shared resource
class SharedData {
    private int value = 0;
    
    public int read() {
        return value;
    }
    
    public void write(int newValue) {
        value = newValue;
    }
}

// Reader thread
class Reader extends Thread {
    private int id;
    private ReaderWriterMonitor monitor;
    private SharedData data;
    
    public Reader(int id, ReaderWriterMonitor monitor, SharedData data) {
        this.id = id;
        this.monitor = monitor;
        this.data = data;
    }
    
    @Override
    public void run() {
        try {
            for (int i = 0; i < 3; i++) {
                Thread.sleep((long)(Math.random() * 1000)); // Random delay
                
                monitor.startRead();
                
                // Critical section - reading
                int value = data.read();
                System.out.println("Reader " + id + ": reading value = " + value);
                Thread.sleep(500); // Simulate reading time
                
                monitor.endRead();
                
                System.out.println("Reader " + id + ": finished reading");
            }
        } catch (InterruptedException e) {
            e.printStackTrace();
        }
    }
}

// Writer thread
class Writer extends Thread {
    private int id;
    private ReaderWriterMonitor monitor;
    private SharedData data;
    private static int nextValue = 1;
    
    public Writer(int id, ReaderWriterMonitor monitor, SharedData data) {
        this.id = id;
        this.monitor = monitor;
        this.data = data;
    }
    
    @Override
    public void run() {
        try {
            for (int i = 0; i < 2; i++) {
                Thread.sleep((long)(Math.random() * 1000)); // Random delay
                
                monitor.startWrite();
                
                // Critical section - writing
                int newValue = nextValue++;
                data.write(newValue);
                System.out.println("Writer " + id + ": wrote value = " + newValue);
                Thread.sleep(500); // Simulate writing time
                
                monitor.endWrite();
                
                System.out.println("Writer " + id + ": finished writing");
            }
        } catch (InterruptedException e) {
            e.printStackTrace();
        }
    }
}

// Main class
public class ReaderWriterDemo {
    public static void main(String[] args) {
        ReaderWriterMonitor monitor = new ReaderWriterMonitor();
        SharedData data = new SharedData();
        
        System.out.println("Starting Reader-Writer Simulation");
        System.out.println("===================================\n");
        
        // Create reader threads
        Reader[] readers = new Reader[3];
        for (int i = 0; i < 3; i++) {
            readers[i] = new Reader(i + 1, monitor, data);
            readers[i].start();
        }
        
        // Create writer threads
        Writer[] writers = new Writer[2];
        for (int i = 0; i < 2; i++) {
            writers[i] = new Writer(i + 1, monitor, data);
            writers[i].start();
        }
        
        // Wait for all threads to complete
        try {
            for (int i = 0; i < 3; i++) {
                readers[i].join();
            }
            for (int i = 0; i < 2; i++) {
                writers[i].join();
            }
        } catch (InterruptedException e) {
            e.printStackTrace();
        }
        
        System.out.println("\n===================================");
        System.out.println("All threads completed. Final value: " + data.read());
    }
}
