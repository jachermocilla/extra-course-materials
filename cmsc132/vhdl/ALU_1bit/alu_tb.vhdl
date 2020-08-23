LIBRARY ieee;
USE ieee.std_logic_1164.all;

ENTITY alu_tb IS
END alu_tb;

ARCHITECTURE behavior OF alu_tb IS
   COMPONENT alu IS
      PORT (a, b, CarryIn, S0, S1: IN STD_LOGIC;
      Result, CarryOut: OUT STD_LOGIC);
   END COMPONENT;
   SIGNAL input: STD_LOGIC_VECTOR(4 DOWNTO 0);
   SIGNAL res, cout: STD_LOGIC;
BEGIN 
   UUT: alu PORT MAP (
      S0 => input(0),
      S1 => input(1),
      CarryIn => input(2),
      a => input(3),
      b => input(4),
      Result => res,
      CarryOut => cout
   );
   STIM_PROC: PROCESS
   BEGIN

      -- 1 AND 1
      input <= "00011"; WAIT FOR 10 NS; ASSERT res='1' REPORT "00011 failed,output= " & STD_LOGIC'IMAGE(res);

      -- 1 OR 1
      input <= "01011"; WAIT FOR 10 NS; ASSERT res='1' REPORT "01011 failed,output= " & STD_LOGIC'IMAGE(res);

      -- 1 + 1
      input <= "10011"; WAIT FOR 10 NS; ASSERT res='0' REPORT "01011 failed,output= " & STD_LOGIC'IMAGE(res);
      WAIT;


   END PROCESS;
END;      
   

