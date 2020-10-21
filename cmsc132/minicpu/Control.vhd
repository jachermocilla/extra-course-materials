library ieee;
use ieee.std_logic_1164.all;

entity Control is
  port(
    instr   : in std_logic_vector(1 downto 0);   -- instruction

    alu_op  : out std_logic_vector(1 downto 0);  -- operation code of AlU
    alu_src : out std_logic;                     -- ALU select ADDi
    reg_dst : out std_logic                      -- select destination address register
  );
end Control;


architecture dataflow of Control is

begin
  with instr select
    alu_op <= "00" when "00",   -- AND
              "01" when "01",   -- OR
              "10" when "10",   -- ADD
              "11" when "11",   -- ADDi
              "XX" when others;   -- error

  with instr select
    alu_src <= '1' when "11",   -- ADDi
               '0' when others;

  with instr select
    reg_dst <= '1' when "11",   -- ADDi
               '0' when others;

end dataflow;
