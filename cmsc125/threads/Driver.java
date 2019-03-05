class Sum
{
   private int sum;

   public int getSum() {
      return sum;
   }
   public void setSum(int sum) {
      this.sum = sum;
   }
}

class Summation implements Runnable
{
   private int upper;
   private Sum sumValue;
   public Summation(int upper, Sum sumValue) {
      this.upper = upper;
      this.sumValue = sumValue;
   }

   public void run() {
      int sum = 0;
      for (int i = 0; i <= upper; i++)
         sum += i;
      sumValue.setSum(sum);
   }
}

public class Driver
{
   public static void main(String[] args) {
   if (args.length > 0) {
      if (Integer.parseInt(args[0]) < 0)
         System.err.println(args[0] + " must be >= 0.");
      else {
         Sum sumObject = new Sum();
         int upper = Integer.parseInt(args[0]);
         Thread thrd = new Thread(new Summation(upper, sumObject));
         thrd.start();
      
         try {
            thrd.join();
            System.out.println("The sum of "+upper+" is "+sumObject.getSum());
         } catch (InterruptedException ie) { }
      }
   }
   else
   System.err.println("Usage: Summation <integer value>"); }
}
