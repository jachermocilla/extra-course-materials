LIBRARY ieee;
USE ieee.std_logic_1164.all;

ENTITY mux_4bit_tb IS
END mux_4bit_tb;

ARCHITECTURE behavior OF mux_4bit_tb IS
   COMPONENT mux_4bit IS 
      PORT (A, B: IN STD_LOGIC_VECTOR(3 DOWNTO 0);
         S: IN STD_LOGIC;
         C: OUT STD_LOGIC_VECTOR(3 DOWNTO 0));
   END COMPONENT;
   SIGNAL input_A: STD_LOGIC_VECTOR(3 DOWNTO 0);
   SIGNAL input_B: STD_LOGIC_VECTOR(3 DOWNTO 0);
   SIGNAL input_S: STD_LOGIC;
   SIGNAL output_C: STD_LOGIC_VECTOR(3 DOWNTO 0);

BEGIN 
   uut: mux_4bit PORT MAP (
      S => input_S,
      A => input_A,
      B => input_B,
      C => output_C
   );

   stim_proc: PROCESS
   BEGIN
      input_A <= "0011";
      input_B <= "1000";
      input_S <= '0';
      WAIT FOR 10 ns;
      ASSERT output_C="0011";   
      
      input_A <= "0011";
      input_B <= "1000";
      input_S <= '1';
      WAIT FOR 10 ns;
      ASSERT output_C="1000";   

      WAIT;
      
   END PROCESS;
END;      
   

