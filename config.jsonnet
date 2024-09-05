local getEnv = function(key, fallback="")
  if std.objectHas(std.parseJson(std.extVar("env")), key) then 
    std.parseJson(std.extVar("env"))[key]
  else
    fallback
  ;

{
  
}
