//http://www.baeldung.com/java-dijkstra
//used in: https://www.codechef.com/status/HOMDEL,jachermocilla
//obviously, TLE :)

import java.util.*;

class Vertex{
   public String name;
   public List<Vertex> sp=new LinkedList<>();
   public Integer d = Integer.MAX_VALUE;
   public Map<Vertex,Integer> adj = new HashMap<>();
   
   public void add(Vertex v, int d){
      adj.put(v,d);
   }

   public Vertex(String name){
      this.name=name;
   } 
   
   public String toString(){
      return name;
   } 
}


class Graph{
   public Set<Vertex> V=new HashSet<>();

   public void add(Vertex v){
      V.add(v);
   }
   
   public void print(){
      for (Vertex u: V){
         System.out.print(u+":");
         for (Map.Entry<Vertex, Integer> pair: u.adj.entrySet()) {
            Vertex v = pair.getKey();
            Integer d = pair.getValue();
            System.out.print("("+v+","+d+")");
         }
         System.out.println();
      }
   }

   public void dijkstra(Vertex source) {
      source.d=0;
      
      Set<Vertex> done = new HashSet<>();
      Queue<Vertex> notDone = new PriorityQueue<>(
                              100000,
                              new Comparator<Vertex>(){
                                 public int compare(Vertex v1, Vertex v2){
                                    return (v1.d-v2.d);
                                 }
                              }
                           ); 
 
      notDone.add(source);
      while (notDone.size() != 0) {
         Vertex u = notDone.poll();
         for (Map.Entry<Vertex, Integer> pair: u.adj.entrySet()) {
            Vertex v = pair.getKey();
            Integer d = pair.getValue();
            if (!done.contains(v)) {
               if (u.d + d < v.d) {
                  v.d = u.d + d;
                  LinkedList<Vertex> sp = new LinkedList<>(u.sp);
                  sp.add(u);
                  v.sp=sp;
               }
               notDone.add(v);
            }
         }
         done.add(u);
      }
   }
   
   public void printSP(){
      for (Vertex v:V){
         System.out.println(v.name+":"+v.d);
         for (Vertex a: v.sp){
            System.out.print(a.name+"->");
         }
         System.out.println(v.name);
      } 
   }

}


public class DIJKSTRA{
   public static void main(String args[]){
      
      final long startTime = System.currentTimeMillis();

      Vertex A = new Vertex("A");
      Vertex B = new Vertex("B");
      Vertex C = new Vertex("C");
      Vertex D = new Vertex("D"); 
      Vertex E = new Vertex("E");
      Vertex F = new Vertex("F");
      Vertex G = new Vertex("G");
   

      A.add(B,2);
      A.add(D,3);

      B.add(A,2);
	   B.add(E,4);
	   B.add(C,5);

	   C.add(B,5);
	   C.add(F,4);
	   C.add(G,3);

	   D.add(A,3);
	   D.add(E,5);

	   E.add(B,4);
	   E.add(D,5);
	   E.add(F,2);

	   F.add(C,4);
	   F.add(G,1);
	   F.add(E,2);

	   G.add(C,3);
	   G.add(F,1); 

      Graph g=new Graph();

      g.add(A);
      g.add(B);
      g.add(C);
      g.add(D);
      g.add(E);
      g.add(F);
      g.add(G);

      System.out.println("----Adjacency List of Input Graph----"); 
      g.print();

      System.out.println("----Shortest Paths rooted at A----"); 
      g.dijkstra(A);
      g.printSP();

      final long endTime = System.currentTimeMillis();
      System.out.println("Total execution time: " + (endTime - startTime)+" ms" );

   }
}
