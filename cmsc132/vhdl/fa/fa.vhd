LIBRARY IEEE;
USE IEEE.STD_LOGIC_1164.ALL;
--------------------------- 
ENTITY full_adder is
   PORT ( a : IN STD_LOGIC;
      b : IN STD_LOGIC;
      CarryIn : IN STD_LOGIC;
      Sum : OUT STD_LOGIC;
      CarryOut : OUT STD_LOGIC);
END full_adder;
--------------------------- 
ARCHITECTURE gate_level OF full_adder IS
BEGIN
 Sum <= a XOR b XOR CarryIn ;
 CarryOut <= (a AND b) OR (CarryIn AND a) 
               OR (CarryIn AND b) ;
end gate_level;
