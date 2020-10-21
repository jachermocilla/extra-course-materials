library ieee;
use ieee.std_logic_1164.all;
use ieee.std_logic_unsigned.all;

entity ALU is
  port(
    op  : in std_logic_vector(1 downto 0);   -- operation code

    rs  : in std_logic_vector(7 downto 0);   -- source register 1
    rt  : in std_logic_vector(7 downto 0);   -- source register 2
    rd  : out std_logic_vector(7 downto 0)   -- destination register
  );
end ALU;


architecture behavioral of ALU is

  signal result : std_logic_vector(7 downto 0);

begin
  process(op, rs, rt)
  begin
    if (op = "00") then      -- AND
      result <= rs and rt;
    elsif (op = "01") then   -- ADD
      result <= rs + rt;
    elsif (op = "10") then   -- SUB
      result <= rs - rt;
    elsif (op = "11") then   -- ADDi
      result <= rs + rt;
    end if;
  end process;

  rd <= result;

end behavioral;