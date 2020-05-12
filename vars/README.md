# howice

## Import packages e.g

```
git clone --branch master --origin origin --progress -v git@github.com:digaverse/howi.git
cd howi
git remote rm origin
git filter-branch --subdirectory-filter lib/vars -- --all
git reset --hard
git gc --aggressive
git prune
git clean -fd
mkdir vars
mv * vars
git add .
git commit
cd ../howice/
git remote add import <local-pah>
git pull import master --allow-unrelated-histories
git remote rm import
git push

```
