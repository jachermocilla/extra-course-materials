LIBRARY ieee;
USE ieee.std_logic_1164.all;

ENTITY alu_tb IS
END alu_tb;

ARCHITECTURE behavior OF alu_tb IS
   COMPONENT alu IS
      PORT (a, b, CarryIn: IN STD_LOGIC;
      Operation: IN STD_LOGIC_VECTOR(1 DOWNTO 0);
      Result, CarryOut: OUT STD_LOGIC);
   END COMPONENT;
   SIGNAL input: STD_LOGIC_VECTOR(4 DOWNTO 0);
   SIGNAL res, cout: STD_LOGIC;
BEGIN 
   UUT: alu PORT MAP (
      Operation(0) => input(4),
      Operation(1) => input(3),
      CarryIn => input(2),
      a => input(1),
      b => input(0),
      Result => res,
      CarryOut => cout
   );
   STIM_PROC: PROCESS
   BEGIN

      -- 1 AND 1
      input <= "00011"; WAIT FOR 10 NS; ASSERT res='1' REPORT "00011 failed,res= " & STD_LOGIC'IMAGE(res);

      -- 1 OR 1
      input <= "01011"; WAIT FOR 10 NS; ASSERT res='1' REPORT "01011 failed,res= " & STD_LOGIC'IMAGE(res);

      -- 1 + 1
      input <= "10011"; WAIT FOR 10 NS; ASSERT res='0' REPORT "10011 failed,res= " & STD_LOGIC'IMAGE(res);
      WAIT;


   END PROCESS;
END;      
   

