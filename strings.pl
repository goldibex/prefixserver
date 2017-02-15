my @chars = ("A".."Z", "a".."z", "0".."9");
my $string;
for (my $i = 0; $i < 2000000; $i++) {
  my $underscore_count = int(rand(3));
  $string = "";
  for (my $j = 0; $j <= $underscore_count; $j++) {
    $string .= $chars[rand @chars] for 1..int(rand(14)+2);
    $string .= "_";
  }
  chop $string;
  $priority = int(rand(100000));
  print "$string $priority\n";
}
