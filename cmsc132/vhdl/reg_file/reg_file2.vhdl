LIBRARY ieee;
USE ieee.std_logic_1164.ALL;
USE IEEE.NUMERIC_STD.ALL;

ENTITY reg_file2 is
   PORT( 
      read_n1: IN STD_LOGIC;
      read_n2: IN STD_LOGIC;
      write_n: IN STD_LOGIC;
      write_data: IN STD_LOGIC;
      write: IN STD_LOGIC;
      clk: IN STD_LOGIC;
      read_data1: OUT STD_LOGIC;
      read_data2: OUT STD_LOGIC);
END reg_file2;
ARCHITECTURE behavioral OF reg_file2 IS
TYPE rf_type IS ARRAY(0 to 1) of std_logic;
SIGNAL registers : rf_type;
BEGIN
   rf: PROCESS(clk)
   BEGIN
      IF RISING_EDGE(clk) THEN
         IF (write ='1') THEN
            registers(to_integer(unsigned'('0' & write_n))) <= write_data;
         END IF;
      END IF;
      read_data1 <= registers(to_integer(unsigned'( '0' & read_n1)));
      read_data2 <= registers(to_integer(unsigned'( '0' & read_n2)));
   END PROCESS;
END behavioral;
