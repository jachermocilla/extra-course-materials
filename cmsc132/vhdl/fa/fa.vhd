LIBRARY IEEE;
USE IEEE.STD_LOGIC_1164.ALL;
 
ENTITY full_adder is
 PORT ( a : IN STD_LOGIC;
 b : IN STD_LOGIC;
 CarryIn : IN STD_LOGIC;
 Sum : OUT STD_LOGIC;
 Cout : OUT STD_LOGIC);
END full_adder;
 
ARCHITECTURE gate_level OF full_adder IS
 
BEGIN
 S <= A XOR B XOR Cin ;
 Cout <= (A AND B) OR (Cin AND A) OR (Cin AND B) ;
end gate_level;
